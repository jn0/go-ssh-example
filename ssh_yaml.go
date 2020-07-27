package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/fslock"

	"gopkg.in/yaml.v2"
)

const LOCK_TIMEOUT = 500 * time.Millisecond

type Job struct {
	lock     *fslock.Lock
	Filename string
	Title    string   `yaml:"title"`   // job title
	Command  string   `yaml:"command"` // job command
	CheckFor string   `yaml:"check"`   // find this in output, optional
	UseTty   bool     `yaml:"tty"`     // request ssh tty, optional
	Domain   string   `yaml:"domain"`  // domain suffix for <hosts>
	User     string   `yaml:"user"`    // ssh user, normally absent
	Before   string   `yaml:"before"`  // setup command, optional
	After    string   `yaml:"after"`   // cleanup command, optional
	Hosts    []string `yaml:"hosts"`   // list of hosts to run the <command> on
}

func (j *Job) Lock() {
	if j.lock == nil {
		j.lock = fslock.New(j.Filename)
	}
	err := j.lock.LockWithTimeout(LOCK_TIMEOUT)
	if err != nil {
		log.Fatal("Cannot lock %q: %v", j.Filename, err)
	}
}
func (j *Job) Unlock() {
	err := j.lock.Unlock()
	if err != nil {
		log.Fatal("Cannot unlock %q: %v", j.Filename, err)
	}
}

func (j *Job) Check(text string) bool {
	return j.CheckFor == "" || strings.Contains(text, j.CheckFor)
}

func (j *Job) Fqdn(name string) string {
	dom := ""
	if j.Domain != "" {
		if !strings.HasPrefix(j.Domain, ".") {
			dom = "."
		}
		dom += j.Domain
	}
	return name + dom
}

func (j *Job) View(show func(string)) {
	show("# JOB FILE " + j.Filename + " #")
	show("title: " + j.Title)

	if j.Before != "" {
		show("before: " + j.Before)
	}
	show("command: " + j.Command)
	if j.After != "" {
		show("after: " + j.After)
	}

	if j.UseTty {
		show("tty: true")
	}
	if j.User != "" {
		show("user: " + j.User)
	}

	if j.CheckFor != "" {
		show("check: " + j.CheckFor)
	}

	if j.Domain != "" {
		show("domain: " + j.Domain)
	}
	show("hosts:")
	for _, h := range j.Hosts {
		show("\t- " + h)
	}
	show("# EOF " + j.Filename + " #")
}

func DirExists(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func FileExists(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && !fi.IsDir()
}

func ListYaml(dir string, show func(string, string)) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if FileExists(path) && strings.HasSuffix(path, ".yaml") {
			j, e := LoadYaml(path, "")
			if e != nil {
				panic(e) // should NOT happen
			}
			show(path, j.Title)
		}
		return nil
	})
}

func YamlFile(name, deflt string) string {
	if FileExists(name) {
		return name
	}
	if !DirExists(deflt) {
		return ""
	}
	name = filepath.Join(deflt, name)
	if !FileExists(name) {
		name += ".yaml"
	}
	if !FileExists(name) {
		return ""
	}
	return name
}

func LoadYaml(name, deflt string) (*Job, error) {

	name = YamlFile(name, deflt)
	if name == "" {
		return nil, os.ErrNotExist
	}
	log.Debug("Will read %q", name)

	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	var job Job
	err = yaml.Unmarshal(data, &job)
	if err != nil {
		return nil, err
	}
	job.Filename = name

	return &job, nil
}
