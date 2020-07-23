package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type Job struct {
	Title   string   `yaml:"title"`
	Command string   `yaml:"command"`
	UseTty  bool     `yaml:"tty"`
	Domain  string   `yaml:"domain"`
	Hosts   []string `yaml:"hosts"`
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

	dom := ""
	if job.Domain != "" {
		if !strings.HasPrefix(job.Domain, ".") {
			dom = "."
		}
		dom += job.Domain
	}

	for i, host := range job.Hosts {
		job.Hosts[i] = host + dom
	}

	return &job, nil
}
