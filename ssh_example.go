/* go build && ./go-ssh-example test.yaml
 * where test.yaml looks like
 ########################################
command: cd /tmp && pwd; exit 0
tty: false
domain: example.com
hosts:
        - test-web
        - test-db
        - test-backup
 ########################################
 * it will run 3 parallel SSH to perform
 * `cd /tmp && pwd; exit 0` on each of
 *      - test-web.example.com
 *      - test-db.example.com
 *      - test-backup.example.com
*/
package main

import (
	"flag"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	logging "github.com/jn0/go-log"
	"github.com/logrusorgru/aurora"
	"gopkg.in/yaml.v2"
)

var log = logging.Root

type Job struct {
	Command string   `yaml:"command"`
	Domain  string   `yaml:"domain"`
	Hosts   []string `yaml:"hosts"`
	UseTty  bool     `yaml:"tty"`
}

var Config struct {
	LogLevel string
	UsePanic bool
	NoColor  bool
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

func run(wg *sync.WaitGroup, task int, context map[string]string, command string) {
	log.Info("[%d] @%q: %q", task, context["host"], command)
	t1 := time.Now()
	out, err := RunCommandOverSsh(context, command)
	t2 := time.Now()
	e := color.Green("ok")
	f := log.Info
	if err != nil {
		e = color.Red(err.Error())
		f = log.Warn
	}
	dt := t2.Sub(t1)
	elapse(task, dt, err)
	f("[%d] @%q: %q (%v) %s", task, context["host"], out, e, dt)
	wg.Done()
}

func main() {
	flag.BoolVar(&Config.UsePanic, "log-panic", false, "use panic() for fatals")
	flag.StringVar(&Config.LogLevel, "log-level", "INFO", "log level")
	flag.BoolVar(&Config.NoColor, "log-color", false, "disable log colors")
	flag.Parse()
	defer log.Debug("Done")

	color = aurora.NewAurora(!Config.NoColor)

	log.SetLevel(logging.LogLevelByName(strings.ToUpper(Config.LogLevel)))
	log.UsePanic(Config.UsePanic)

	task := 0
	elapsed = make(map[int]time.Duration)
	result = make(map[int]error)
	wg := sync.WaitGroup{}
	t1 := time.Now()
	for _, arg := range flag.Args() {
		data, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Error("Cannot read %q: %v", arg, err)
			continue
		}

		var job Job
		err = yaml.Unmarshal(data, &job)
		if err != nil {
			log.Error("Cannot process %q: %v", arg, err)
		}

		dom := ""
		if job.Domain != "" {
			if !strings.HasPrefix(job.Domain, ".") {
				dom = "."
			}
			dom += job.Domain
		}

		for _, host := range job.Hosts {
			elapsed[task] = 0
			context := NewContext(host+dom, job.UseTty)
			wg.Add(1)
			go run(&wg, task, context, job.Command)
			task += 1
		}
	}
	t2 := time.Now()
	log.Debug("All started in %s", t2.Sub(t1))
	wg.Wait()
	t2 = time.Now()
	failed := 0
	var total time.Duration
	for t, v := range elapsed {
		total += v
		if result[t] != nil {
			failed += 1
		}
	}
	log.Info("Total run time %s for %d tasks in %s (%.1f× speedup)",
		total, len(elapsed), t2.Sub(t1), total.Seconds()/t2.Sub(t1).Seconds())
	if failed != 0 {
		log.Warn("There were %d failed tasks out of %d, %.0f%%",
			failed, len(result), float64(100*failed)/float64(len(result)))
	}
}
