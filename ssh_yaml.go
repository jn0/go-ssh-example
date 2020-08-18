package main

import (
	"fmt"
	"io"
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
	// YAML fillable:
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

func (j *Job) Error(text string, err error) error {
	return fmt.Errorf("Job %q failed in %s: %v", j.Title, text, err)
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

	var text_or_comment = func(name, value, stub string) {
		if value != "" {
			show(Config.NameColor(name) + Config.DivColor(": ") + value)
		} else {
			show(Config.CommentColor("# " + name + ": " + stub))
		}
	}
	var bool_or_comment = func(name string, value bool) {
		if value {
			show(Config.NameColor(name) + Config.DivColor(": ") + "true")
		} else {
			show(Config.CommentColor("# " + name + ": false"))
		}
	}

	show(Config.CommentColor("# JOB FILE " + j.Filename + " #"))
	text_or_comment("title", j.Title, strings.Title(strings.TrimSuffix(filepath.Base(j.Filename), ".yaml")))
	text_or_comment("before", j.Before, "/bin/true")
	text_or_comment("command", j.Command, "/bin/false")
	text_or_comment("after", j.After, "/bin/true")

	bool_or_comment("tty", j.UseTty)
	text_or_comment("user", j.User, "<current user>")

	text_or_comment("check", j.CheckFor, "<nothing special>")

	text_or_comment("domain", j.Domain, "example.com")
	show(Config.NameColor("hosts") + Config.DivColor(":"))
	for _, h := range j.Hosts {
		show(Config.DivColor("    - ") + h)
	}
	show(Config.CommentColor("# EOF " + j.Filename + " #"))
}

func DirExists(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func DirCreate(name string) {
	if !DirExists(name) {
		err := os.MkdirAll(name, 0750)
		if err != nil {
			log.Fatal("Cannot make directory %q: %v", name, err)
		}
		log.Info("The %q has been created", name)
	}
}

func CreateYaml(name string) {
	title := strings.Title(
		strings.Replace(
			strings.TrimSuffix(filepath.Base(name), ".yaml"),
			"_", " ", -1))
	text := "## sample JOB FILE " + name + " template #\n"
	text += "#title: " + title + "\n"
	text += "#before: /bin/true\n"
	text += "#command: /bin/false\n"
	text += "#after: /bin/true\n"
	text += "#tty: false\n"
	text += "#user: <current user>\n"
	text += "#check: <text to search for>\n"
	text += "#domain: <domain name to append to hostnames>\n"
	text += "#hosts:\n"
	text += "#    - host1\n"
	text += "#    - host2\n"
	text += "#    - host3\n"
	text += "## EOF " + name + " #\n"
	err := ioutil.WriteFile(name, []byte(text), 0640)
	if err != nil {
		log.Fatal("Cannot create %q: %v", name, err)
	}
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

func CopyYaml(dst, src string) {
	sf, err := os.Stat(src)
	if err != nil {
		log.Fatal("Copy: cannot stat %q: %v", src, err)
	}
	if !sf.Mode().IsRegular() {
		log.Fatal("Copy: source %q is not a regular file (%s)", src, sf.Mode())
	}
	if FileExists(dst) {
		log.Fatal("Copy: target file %q exists", dst)
	}
	fs, err := os.Open(src)
	if err != nil {
		log.Fatal("Copy: cannot open %q: %v", src, err)
	}
	defer fs.Close()
	fd, err := os.Create(dst)
	if err != nil {
		log.Fatal("Copy: cannot create %q: %v", dst, err)
	}
	defer fd.Close()
	written, err := io.Copy(fd, fs)
	if err != nil {
		log.Fatal("Copy: cannot copy %q to %q: %v", src, dst, err)
	}
	err = os.Chmod(dst, 0640)
	if err != nil {
		log.Fatal("Copy: cannot chmod %q: %v", dst, err)
	}
	log.Debug("Copy(%q -> %q): %v bytes", src, dst, written)
}

func NewYamlFileName(name, deflt string) string {
	yaml := filepath.Join(deflt, name)
	if !strings.HasSuffix(yaml, ".yaml") {
		yaml += ".yaml"
	}
	return yaml
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
