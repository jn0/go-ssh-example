package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type Job struct {
	Title    string   `yaml:"title"`
	Command  string   `yaml:"command"`
	CheckFor string   `yaml:"check"`
	UseTty   bool     `yaml:"tty"`
	Domain   string   `yaml:"domain"`
	User     string   `yaml:"user"`
	Hosts    []string `yaml:"hosts"`
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
	show("# JOB FILE #")
	show("title: " + j.Title)
	show("command: " + j.Command)
	if j.CheckFor != "" {
		show("check: " + j.CheckFor)
	}
	if j.UseTty {
		show("tty: true")
	}
	if j.Domain != "" {
		show("domain: " + j.Domain)
	}
	show("hosts:")
	for _, h := range j.Hosts {
		show("\t- " + h)
	}
	show("# EOF #")
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

func LoadYaml(name, deflt string) (*Job, error) {

	if !FileExists(name) {
		if !DirExists(deflt) {
			return nil, os.ErrNotExist
		}
		name = filepath.Join(deflt, name)
		if !FileExists(name) {
			name += ".yaml"
		}
		if !FileExists(name) {
			return nil, os.ErrNotExist
		}
		log.Debug("Will read %q", name)
	}

	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	var job Job
	err = yaml.Unmarshal(data, &job)
	if err != nil {
		return nil, err
	}

	return &job, nil
}
