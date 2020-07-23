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
}

var elapsed map[int]time.Duration

func run(wg *sync.WaitGroup, task int, context map[string]string, command string) {
	log.Info("[%d] @%q: %q", task, context["host"], command)
	t1 := time.Now()
	out, err := RunCommandOverSsh(context, command)
	t2 := time.Now()
	e := aurora.Green("ok")
	if err != nil {
		e = aurora.Red(err.Error())
	}
	dt := t2.Sub(t1)
	elapsed[task] = dt
	log.Info("[%d] @%q: %q (%v) %s", task, context["host"], out, e, dt)
	wg.Done()
}

func main() {
	flag.BoolVar(&Config.UsePanic, "log-panic", false, "use panic() for fatals")
	flag.StringVar(&Config.LogLevel, "log-level", "INFO", "log level")
	flag.Parse()
	defer log.Debug("Done")

	log.SetLevel(logging.LogLevelByName(strings.ToUpper(Config.LogLevel)))
	log.UsePanic(Config.UsePanic)

	task := 0
	elapsed = make(map[int]time.Duration)
	wg := sync.WaitGroup{}
	t1 := time.Now()
	for _, arg := range flag.Args() {
		data, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Fatal("Cannot read %q: %v", arg, err)
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
	var total time.Duration
	for _, v := range elapsed {
		total += v
	}
	log.Info("Total run time %s for %d tasks in %s (%.1f× speedup)", total, len(elapsed), t2.Sub(t1), total.Seconds()/t2.Sub(t1).Seconds())
}
