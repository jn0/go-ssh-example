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
	"flag"
	"os"
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
	ListDir    bool
	UsePanic   bool
	NoColor    bool
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

func run(wg *sync.WaitGroup, context *Context, job *Job) {
	log.Info("[%d] @%q: %q", context.Id, context.Host, job.Command)
	t1 := time.Now()
	out, err := context.Run(job.Command)
	t2 := time.Now()
	e := color.Green("ok")
	ok := true
	f := log.Info
	if err != nil {
		e = color.Red(err.Error())
		f = log.Warn
		ok = false
	}
	dt := t2.Sub(t1)
	elapse(context.Id, dt, err)
	if !job.Check(out) {
		e = color.BrightYellow("output check failed")
		f = log.Warn
		ok = false
	}
	f("[%d] @%q: %v, %s", context.Id, context.Host, e, dt)
	if !ok {
		show_output(context.Id, context.Host, out)
	}
	wg.Done()
}

func main() {
	var err error
	Config.DefaultDir, err = os.UserConfigDir()
	if err != nil {
		Config.DefaultDir = "/etc/rupdate.d"
	} else {
		Config.DefaultDir = filepath.Join(Config.DefaultDir, "rupdate.d")
	}
	flag.StringVar(&Config.DefaultDir, "dir", Config.DefaultDir,
		"default directory for yaml scripts")
	flag.BoolVar(&Config.ListDir, "list", false, "list the <dir> or its entry")

	flag.BoolVar(&Config.UsePanic, "log-panic", false, "use panic() for fatals")
	flag.StringVar(&Config.LogLevel, "log-level", "INFO", "log level")
	flag.BoolVar(&Config.NoColor, "log-color", false, "disable log colors")

	flag.Parse()
	defer log.Debug("Done")

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

	color = aurora.NewAurora(!Config.NoColor)

	log.SetLevel(logging.LogLevelByName(strings.ToUpper(Config.LogLevel)))
	log.UsePanic(Config.UsePanic)

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

		for _, host := range job.Hosts {
			elapsed[task] = 0
			wg.Add(1)
			go run(&wg, NewContext(task, job.Fqdn(host), job.UseTty), job)
			task += 1
		}
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
