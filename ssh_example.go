/* go build && ./go-ssh-example test.yaml
 * where test.yaml looks like
 ########################################
command: cd /tmp && pwd; tty
tty: false
domain: example.com
hosts:
        - test-web
        - test-db
        - test-backup
 ########################################
 * it will run 3 parallel SSH to perform
 * `cd /tmp && pwd; tty` on each of
 *      - test-web.example.com
 *      - test-db.example.com
 *      - test-backup.example.com
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	logging "github.com/jn0/go-log"
	"github.com/logrusorgru/aurora"
)

var log = logging.Root

var Config struct {
	LogLevel   string
	DefaultDir string
	SaveDir    string
	ListDir    bool
	UsePanic   bool
	NoColor    bool
	Edit       bool
}

var color aurora.Aurora
var lock_elapsed sync.Mutex
var elapsed map[int]time.Duration
var result map[int]error

func elapse(task int, d time.Duration, e error) {
	lock_elapsed.Lock()
	defer lock_elapsed.Unlock()
	elapsed[task] = d // it has to be locked. real shit.
	result[task] = e
}

func totals() (failed int, total time.Duration) {
	for t, v := range elapsed {
		total += v
		if result[t] != nil {
			failed += 1
		}
	}
	return
}

func show_output(task int, host, text string) {
	lines := strings.Split(text, "\n")
	nl := len(lines)
	digits := 0
	for nl > 0 {
		digits += 1
		nl /= 10
	}
	prefix := host + ":" + strconv.FormatInt(int64(task), 10) + ":"
	for _, line := range lines {
		os.Stdout.Write([]byte(prefix + line + "\n"))
	}
}

func save_output(context *Context, command, out string) {
	if Config.SaveDir == "" {
		return
	}
	fname := filepath.Join(Config.SaveDir, context.Host)
	data := "# Host:    " + context.Host + "\n" +
		"# Command: " + command + "\n" +
		"# User:    " + context.User + " (" + context.Gecos + ")\n" +
		"# Started: " + context.Time.Start.String() + "\n" +
		"# Ended:   " + context.Time.Stop.String() + "\n" +
		"# Elapsed: " + context.Time.Stop.Sub(context.Time.Start).String() + "\n" +
		"\n" + out + "\n### EOF ###\n"
	err := ioutil.WriteFile(fname, []byte(data), 0640)
	if err != nil {
		log.Error("[%d] Cannot save %q: %v", context.Id, fname, err)
	}
}

func run(wg *sync.WaitGroup, context *Context, job *Job) {
	log.Info("[%d] @%q: %q", context.Id, context.Host, job.Command)
	t1 := time.Now()
	out, err := context.Run(job.Command)
	t2 := time.Now()
	e := color.Green("ok")
	ok := true
	f := log.Info
	if ok && err != nil {
		e = color.Red(err.Error())
		f = log.Warn
		ok = false
	}
	dt := t2.Sub(t1)
	elapse(context.Id, dt, err)
	if ok && !job.Check(out) {
		e = color.BrightYellow("output check failed")
		f = log.Warn
		ok = false
	}
	f("[%d] @%q: %v, %s", context.Id, context.Host, e, dt)
	save_output(context, job.Command, out)
	if !ok {
		show_output(context.Id, context.Host, out)
	}
	wg.Done()
}

func bash(args ...string) error {
	cmd := exec.Command("bash", "-c")
	if !FileExists(cmd.Path) {
		return errors.New(fmt.Sprintf("No %q", cmd.Path))
	}
	cmd.Args = append(cmd.Args, strings.Join(args, " "))
	log.Debug("%q %#v", cmd.Path, cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(fmt.Sprintf("Error running %q: %v", cmd.Path, err))
	}
	return nil
}

var envEditorNames = []string{"VISUAL", "EDITOR"}

func _edit(editor, file string) error {
	cmd := exec.Command(editor, file)
	if !FileExists(cmd.Path) {
		return errors.New(fmt.Sprintf("No %q", cmd.Path))
	}
	log.Debug("%q %v", cmd.Path, cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(fmt.Sprintf("Error running %q: %v", cmd.Path, err))
	}
	return nil
}

const TheEditor = "vi"

func edit(name string) {
	if name == "" {
		return
	}
	for _, ev := range envEditorNames {
		ed, ok := os.LookupEnv(ev)
		if !ok || ed == "" {
			continue
		}
		err := _edit(ed, name)
		if err != nil {
			log.Warn("%q=%q: %v", ed, ev, err)
			continue
		}
		return
	}
	if _edit(TheEditor, name) == nil { // last resort
		return
	}
	log.Fatal("No editor found")
}

func main() {
	var err error
	Config.DefaultDir, err = os.UserConfigDir()
	if err != nil {
		Config.DefaultDir = filepath.Join("/etc", filepath.Base(os.Args[0])+".d")
	} else {
		Config.DefaultDir = filepath.Join(Config.DefaultDir, filepath.Base(os.Args[0])+".d")
	}
	flag.StringVar(&Config.DefaultDir, "dir", Config.DefaultDir,
		"default directory for yaml scripts")
	flag.BoolVar(&Config.ListDir, "list", false, "list the <dir> or its entry")

	flag.BoolVar(&Config.UsePanic, "log-panic", false, "use panic() for fatals")
	flag.StringVar(&Config.LogLevel, "log-level", "INFO", "log level")
	flag.BoolVar(&Config.NoColor, "log-color", false, "disable log colors")

	flag.StringVar(&Config.SaveDir, "save", Config.SaveDir, "directory to save output to")
	flag.BoolVar(&Config.Edit, "edit", false, "run editor on the yaml")

	flag.Parse()
	defer log.Debug("Done")

	color = aurora.NewAurora(!Config.NoColor)

	log.SetLevel(logging.LogLevelByName(strings.ToUpper(Config.LogLevel)))
	log.UsePanic(Config.UsePanic)

	if Config.Edit {
		if flag.NArg() == 0 {
			edit(Config.DefaultDir)
		} else {
			for _, arg := range flag.Args() {
				edit(YamlFile(arg, Config.DefaultDir))
			}
		}
		return
	}

	if flag.NArg() == 0 {
		ListYaml(Config.DefaultDir, func(pth, title string) {
			os.Stdout.Write([]byte(strings.TrimSuffix(path.Base(pth), ".yaml") +
				"\t" + title + "\n"))
		})
		return
	}
	if Config.ListDir {
		for _, arg := range flag.Args() {
			job, err := LoadYaml(arg, Config.DefaultDir)
			if err != nil {
				os.Stderr.Write([]byte(arg + ": " + err.Error() + "\n"))
				continue
			}
			job.View(func(line string) {
				os.Stdout.Write([]byte(line + "\n"))
			})
		}
		return
	}

	if Config.SaveDir != "" && !DirExists(Config.SaveDir) {
		log.Fatal("Cannot save to %q: it does not exist", Config.SaveDir)
	}

	var do_the_job = func(task *int, job *Job, wg *sync.WaitGroup) {
		job.Lock()
		defer func() {
			job.Unlock()
			wg.Done()
			err := recover()
			if err != nil {
				log.Fatal("%v", err)
			}
		}()

		if job.Before != "" {
			log.Info("Before %q performing %q", job.Title, job.Before)
			err := bash(job.Before)
			if err != nil {
				panic(errors.New(fmt.Sprintf("Job %q failed in prephase: %v",
					job.Title, err)))
			}
		}

		wgx := sync.WaitGroup{}
		for _, host := range job.Hosts {
			elapsed[*task] = 0
			wgx.Add(1)
			go run(&wgx, NewContext(*task, job.Fqdn(host), job.UseTty, job.User), job)
			*task += 1
		}
		wgx.Wait()

		if job.After != "" {
			log.Info("After %q performing %q", job.Title, job.After)
			err := bash(job.After)
			if err != nil {
				panic(errors.New(fmt.Sprintf("Job %q failed in cleanup: %v",
					job.Title, err)))
			}
		}
	}

	task := 0
	elapsed = make(map[int]time.Duration)
	result = make(map[int]error)
	wg := sync.WaitGroup{}
	t1 := time.Now()
	for _, arg := range flag.Args() {
		job, err := LoadYaml(arg, Config.DefaultDir)
		if err != nil {
			log.Error("Cannot read %q: %v", arg, err)
			continue
		}

		if job.Command == "" || len(job.Hosts) == 0 {
			log.Warn("Nothing to do in %q (%s)", arg, job.Title)
			continue
		}

		wg.Add(1)
		go do_the_job(&task, job, &wg)
	}
	t2 := time.Now()

	log.Debug("All started in %s", t2.Sub(t1))
	wg.Wait()
	t2 = time.Now()

	failed, total := totals()
	log.Info("Total run time %s for %d tasks in %s (%.1f× speedup)",
		total, len(elapsed), t2.Sub(t1), total.Seconds()/t2.Sub(t1).Seconds())
	if failed != 0 {
		log.Warn("There were %d failed tasks out of %d, %.0f%%",
			failed, len(result), float64(100*failed)/float64(len(result)))
	}
}
