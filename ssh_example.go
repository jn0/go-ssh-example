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
	"golang.org/x/sys/unix"
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
	Copy       string
	Create     bool
	//
	Color                                                                         aurora.Aurora
	ErrorColor, FileColor, TitleColor, OkColor, CommentColor, NameColor, DivColor func(s string) string
}

var lock_elapsed sync.Mutex
var elapsed map[int]time.Duration
var result map[int]error

func IsAtty(f *os.File) bool {
	var fd uintptr = f.Fd()
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err == nil
}

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

	f, e, ok := log.Info, Config.OkColor("ok"), true
	if ok && err != nil {
		f, e, ok = log.Warn, err.Error(), false
	}

	dt := t2.Sub(t1)
	elapse(context.Id, dt, err)

	if ok && !job.Check(out) {
		f, e, ok = log.Warn, "output check failed", false
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
	flag.BoolVar(&Config.ListDir, "ls", false, "short for --list")
	flag.BoolVar(&Config.ListDir, "cat", false, "short for --list")
	flag.BoolVar(&Config.ListDir, "l", false, "short for --list")

	flag.BoolVar(&Config.UsePanic, "log-panic", false, "use panic() for fatals")
	flag.StringVar(&Config.LogLevel, "log-level", "INFO", "log level")
	flag.BoolVar(&Config.NoColor, "log-no-color", false, "disable log colors")

	flag.StringVar(&Config.SaveDir, "save", Config.SaveDir, "directory to save output to")
	flag.BoolVar(&Config.Edit, "edit", false, "run editor on the yaml")
	flag.BoolVar(&Config.Edit, "vi", false, "short for --edit")
	flag.BoolVar(&Config.Edit, "e", false, "short for --edit")
	flag.StringVar(&Config.Copy, "copy", "", "create a new yaml from this one")
	flag.StringVar(&Config.Copy, "cp", "", "short for --copy")
	flag.BoolVar(&Config.Create, "create", false, "create a new yaml")
	flag.BoolVar(&Config.Create, "c", false, " short for --create")

	flag.Parse()
	defer log.Debug("Done")

	Config.Color = aurora.NewAurora(IsAtty(os.Stdout) && !Config.NoColor)
	log.UseColor(Config.Color)
	Config.ErrorColor = func(s string) string {
		return Config.Color.BgRed(Config.Color.BrightWhite(s)).String()
	}
	Config.FileColor = func(s string) string { return Config.Color.BrightCyan(s).String() }
	Config.TitleColor = func(s string) string { return Config.Color.Yellow(s).String() }
	Config.OkColor = func(s string) string { return Config.Color.BrightGreen(s).String() }
	Config.CommentColor = func(s string) string { return Config.Color.Cyan(s).String() }
	Config.NameColor = func(s string) string { return Config.Color.BrightYellow(s).String() }
	Config.DivColor = func(s string) string { return Config.Color.Red(s).String() }

	log.SetLevel(logging.LogLevelByName(strings.ToUpper(Config.LogLevel)))
	log.UsePanic(Config.UsePanic)

	if Config.Create {
		if flag.NArg() == 0 {
			log.Fatal("What do you want to create?")
		}
		DirCreate(Config.DefaultDir)
		for _, arg := range flag.Args() {
			yaml := YamlFile(arg, Config.DefaultDir)
			if yaml != "" {
				log.Warn("File %q already exists", yaml)
				continue
			}
			CreateYaml(NewYamlFileName(arg, Config.DefaultDir))
		}
		if !Config.Edit {
			return
		}
	}
	if Config.Copy != "" {
		if flag.NArg() == 0 {
			log.Fatal("Where to copy the %q to?", Config.Copy)
		}
		source := YamlFile(Config.Copy, Config.DefaultDir)
		if source == "" {
			log.Fatal("No source file %q", Config.Copy)
		}
		for _, arg := range flag.Args() {
			CopyYaml(NewYamlFileName(arg, Config.DefaultDir), source)
		}
		if !Config.Edit {
			return
		}
	}
	if Config.Edit {
		if flag.NArg() == 0 {
			DirCreate(Config.DefaultDir)
			Edit(Config.DefaultDir)
		} else {
			for _, arg := range flag.Args() {
				yaml := YamlFile(arg, Config.DefaultDir)
				if yaml == "" {
					DirCreate(Config.DefaultDir)
					yaml = filepath.Join(Config.DefaultDir, arg)
					if !strings.HasSuffix(yaml, ".yaml") {
						yaml += ".yaml"
					}
				}
				Edit(yaml)
			}
		}
		return
	}

	if flag.NArg() == 0 {
		ListYaml(Config.DefaultDir, func(pth, title string) {
			file := strings.TrimSuffix(path.Base(pth), ".yaml")
			os.Stdout.Write([]byte(Config.FileColor(file) +
				"\t" + Config.TitleColor(title) + "\n"))
		})
		return
	}
	if Config.ListDir {
		for _, arg := range flag.Args() {
			job, err := LoadYaml(arg, Config.DefaultDir)
			if err != nil {
				os.Stderr.Write([]byte(Config.ErrorColor(arg+": "+err.Error()) + "\n"))
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
				panic(job.Error("setup", err))
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
				panic(job.Error("cleanup", err))
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
