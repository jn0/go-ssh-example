package main

import (
	"sort"
	"strings"

	"github.com/gobwas/glob"
)

/*============================================================================*/

type SshConfigFileEntry map[string]string // key:value pairs like "Port":"2222"

func (self *SshConfigFileEntry) IsNull() bool {
	return self == nil || *self == nil
}
func (self *SshConfigFileEntry) Has(name string) bool {
	if self == nil || *self == nil {
		return false
	}
	_, ok := (*self)[name]
	return ok
}
func (self *SshConfigFileEntry) Get(name string) (value string, ok bool) {
	if self == nil || *self == nil {
		ok = false
		return
	}
	value, ok = (*self)[name]
	return
}
func (self *SshConfigFileEntry) GetBool(name string) (value bool, ok bool) {
	str, ok := self.Get(name)
	if !ok {
		return false, false
	}
	switch str {
	case "yes":
		return true, true
	case "no":
		return false, true
	case "confirm", "ask":
		return false, true // non-interactive
	}
	return false, false
}
func (self *SshConfigFileEntry) Set(name, value string) *SshConfigFileEntry {
	if *self == nil {
		*self = make(map[string]string)
	}
	(*self)[name] = strings.TrimSpace(value)
	return self
}
func (self *SshConfigFileEntry) String() string {
	var names []string
	for name, _ := range *self {
		names = append(names, name)
	}
	sort.Strings(names)
	var r []string
	for _, name := range names {
		value, ok := self.Get(name)
		if !ok {
			log.Fatal("No entry for %q", name)
		}
		r = append(r, "\t"+name+" "+value)
	}
	return strings.Join(r, "\n")
}

/*============================================================================*/

type SshConfigFile struct {
	name  string                         // path
	hosts []string                       // keep the sequence
	entry map[string]*SshConfigFileEntry // actual data
	globs map[string]glob.Glob           // possible wildcards handling
}

func (self *SshConfigFile) Get(host, sect, dflt string) string {
	res := dflt
	x, h := self.get(host)
	if h != nil {
		v, ok := h.Get(sect)
		if ok {
			res = v
		}
	}
	if host != x {
		x = x + ">" + host
	}
	log.Debug("%q[%q.%q] (%q) = %q", self.name, x, sect, dflt, res)
	return res
}
func (self *SshConfigFile) get(name string) (string, *SshConfigFileEntry) {
	if self == nil { // || *self == nil {
		return name, nil
	}
	ent, ok := self.entry[name]
	if ok {
		return name, ent
	}
	for _, host := range self.hosts {
		if host == name {
			ent, ok = self.entry[name]
			if !ok {
				log.Fatal("%q equals %q, but nothing found",
					name, host)
			}
			return host, ent
		}
		g, ok := self.globs[host]
		if !ok {
			log.Fatal("No entry for %q while looking for %q",
				host, name)
		}
		if g.Match(name) {
			ent, ok = self.entry[host]
			if !ok {
				log.Fatal("%q matched %q, but nothing found",
					name, host)
			}
			return host, ent
		}
	}
	return name, nil
}
func (self *SshConfigFile) Name() string {
	return self.name
}
func (self *SshConfigFile) Len() int {
	return len(self.hosts)
}
func (self *SshConfigFile) Has(name string) bool {
	_, h := self.get(name)
	return h != nil
}
func (self *SshConfigFile) SetName(name string) *SshConfigFile {
	self.name = name
	return self
}
func (self *SshConfigFile) Set(name string, value *SshConfigFileEntry) *SshConfigFile {
	if self.entry == nil {
		self.entry = make(map[string]*SshConfigFileEntry)
		self.globs = make(map[string]glob.Glob)
	}
	self.hosts = append(self.hosts, name)
	g, err := glob.Compile(name)
	if err != nil {
		log.Fatal("Bad glob %q", name)
	}
	self.globs[name] = g
	self.entry[name] = value
	return self
}
func (self *SshConfigFile) String() string {
	var list []string
	for _, host := range self.hosts {
		list = append(list, "Host "+host, self.entry[host].String())
	}
	return "# " + self.name + " #\n" + strings.Join(list, "\n") + "\n# EOF #"
}

/*============================================================================*/

type SshConfig []*SshConfigFile // just a sequence of config files

func (self *SshConfig) Load(name string) {
	cfg, err := LoadSshConfigFile(name)
	if err != nil {
		log.Fatal("SSH config file %q: %v", name, err)
	}
	log.Debug("SSH config file %q has %d host entries", cfg.Name(), cfg.Len())
	*self = append(*self, cfg)
}

func (self *SshConfig) GetValue(host, name, dflt string) (res string) {
	res, _ = self.Get(host, name, dflt)
	return
}

func (self *SshConfig) Get(host, name, dflt string) (res string, found bool) {
	res = dflt
	for _, config := range *self {
		x, h := config.get(host)
		if h == nil {
			continue
		}
		v, ok := h.Get(name)
		if ok {
			found = true
			res = v
		}
		if host != x {
			x = x + ">" + host
		}
		log.Debug("%q[%q.%q] (%q) = %q", config.name, x, name, dflt, res)
		break
	}
	return
}

func NewSshConfig(names ...string) *SshConfig {
	cfg := new(SshConfig)
	if len(names) == 0 {
		names = []string{DefaultSshConfigFile, SystemSshConfigFile}
	}
	for _, name := range names {
		cfg.Load(name)
	}
	return cfg
}

/* EOF */
