package main

import (
	"flag"
	"io/ioutil"
	"strings"
	"sync"

	logging "github.com/jn0/go-log"
	"gopkg.in/yaml.v2"
)

var log = logging.Root

type Job struct {
	Command string   `yaml:"command"`
	Domain  string   `yaml:"domain"`
	Hosts   []string `yaml:"hosts"`
}

var Config struct {
	LogLevel string
	UseTty   bool
	UsePanic bool
}

func run(wg *sync.WaitGroup, context map[string]string, command string) {
	out := RunCommandOverSsh(context, command)
	log.Info("Got %q", out)
	wg.Done()
}

func main() {
	flag.BoolVar(&Config.UseTty, "term", false, "request tty")
	flag.BoolVar(&Config.UsePanic, "log-panic", false, "use panic() for fatals")
	flag.StringVar(&Config.LogLevel, "log-level", "INFO", "log level")
	flag.Parse()
	defer log.Debug("Done")

	log.SetLevel(logging.LogLevelByName(strings.ToUpper(Config.LogLevel)))
	log.UsePanic(Config.UsePanic)

	wg := sync.WaitGroup{}
	for _, arg := range flag.Args() {
		var job Job
		data, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Fatal("Cannot read %q: %v", arg, err)
		}
		err = yaml.Unmarshal(data, &job)
		if err != nil {
			log.Error("Cannot process %q: %v", arg, err)
		}
		for _, host := range job.Hosts {
			if job.Domain != "" {
				host += "." + job.Domain
			}
			log.Info("@ %q", host)
			context := NewContext(host, Config.UseTty)
			wg.Add(1)
			go run(&wg, context, job.Command)
		}
	}
	log.Debug("All started")
	wg.Wait()

}
