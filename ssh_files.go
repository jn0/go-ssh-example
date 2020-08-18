package main

import (
	"io/ioutil"
	"os"
	"os/user"
	fpth "path"
	"path/filepath"
	"regexp"
	"strings"
)

/*============================================================================*/

const (
	SystemSshConfigFile        = "/etc/ssh/ssh_config"
	DefaultSshConfigFile       = ".ssh/config"
	DefaultSshKnownHosts       = ".ssh/known_hosts"
	DefaultSshKeyFile          = ".ssh/id_rsa"
	DefaultSshKeyFilePubSuffix = ".pub"
)

/*============================================================================*/

func getSshUser(name string) (u *user.User, err error) {
	if name != "" {
		u, err = user.Lookup(name)
		if err != nil {
			log.Error("User %q lookup error: %v", name, err)
			return nil, err
		}
	} else {
		u, err = user.Current()
		if err != nil {
			log.Error("Current user lookup error: %v", err)
			return nil, err
		}
	}
	log.Debug("ssh user %q -> %q", name, u.Username)
	return u, nil
}

func findSshFile(username, filename string) (path string, err error) {
	u, err := getSshUser(username)
	if u == nil {
		return
	}
	path = fpth.Join(u.HomeDir, filename)
	_, err = os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		log.Warn("No file %q", path)
		path = ""
	}
	return
}

func FindHostKeyFile(username string) (path string) {
	u, err := getSshUser(username)
	if u == nil {
		return
	}
	path, err = findSshFile(u.Username, DefaultSshKnownHosts)
	if err != nil {
		log.Warn("Cannot find known hosts %q for user %q in %q: %v",
			DefaultSshKnownHosts, u.Username, path, err)
		path = ""
	}
	return
}

func FindSshPubKeyFile(username string) (path string) {
	u, err := getSshUser(username)
	if u == nil {
		return
	}
	path, err = findSshFile(u.Username, DefaultSshKeyFile+DefaultSshKeyFilePubSuffix)
	if err != nil {
		log.Warn("Cannot find pub key %q for user %q in %q: %v",
			DefaultSshKeyFile+DefaultSshKeyFilePubSuffix,
			u.Username, path, err)
		path = ""
	}
	return
}

func FindSshPvtKeyFile(username string) (path string) {
	u, err := getSshUser(username)
	if u == nil {
		return
	}
	path, err = findSshFile(u.Username, DefaultSshKeyFile)
	if err != nil {
		log.Warn("Cannot find key %q for user %q in %q: %v",
			DefaultSshKeyFile, u.Username, path, err)
		path = ""
	}
	return
}

func FindSshConfigFile(username string) (path string) {
	u, err := getSshUser(username)
	if u == nil {
		return
	}
	path, err = findSshFile(u.Username, DefaultSshConfigFile)
	if err != nil {
		log.Warn("Cannot find config %q for user %q in %q: %v",
			DefaultSshConfigFile, u.Username, path, err)
		path = ""
	}
	return
}

/*============================================================================*/
func IsCommentOrBlank(line string) bool {
	const comment = "#"
	return strings.TrimSpace(line) == "" ||
		strings.HasPrefix(strings.TrimSpace(line), comment)
}

/*============================================================================*/

var leadingSpace_re = regexp.MustCompile(`^\s+`)
var spaces_re = regexp.MustCompile(`\s+`)

func LoadSshConfigFile(name string) (cfg *SshConfigFile, e error) {
	const (
		nl = "\n"
		sp = " "
	)

	log.Debug("Loading %q...", name)

	fname := name
	if name == "" || name == DefaultSshConfigFile {
		fname = FindSshConfigFile("")
		if fname == "" {
			log.Warn("No SSH config")
			return
		}
	}

	bytes, e := ioutil.ReadFile(fname)
	if e != nil {
		log.Error("Cannot read %q: %v", name, e)
		return
	}

	var host string
	var entry *SshConfigFileEntry
	for i, line := range strings.Split(string(bytes), nl) {
		// log.Debug("%q[%d]: %q", name, i + 1, line)
		if IsCommentOrBlank(line) {
			continue
		}
		if strings.HasPrefix(line, "Include") {
			// Include /etc/ssh/ssh_config.d/*.conf
			pattern := strings.TrimSpace(line[7:])
			if pattern == "" {
				log.Error("%q[%d]: %q - no value", name, i+1, line)
				continue
			}
			list, e := filepath.Glob(pattern)
			if e != nil {
				log.Error("%q[%d]: %q - bad pattern (%v)", name, i+1, line, e)
				continue
			}
			if len(list) == 0 {
				log.Debug("%q[%d]: %q - no files match pattern %q",
					name, i+1, line, pattern)
				continue
			}
			for _, xname := range list {
				cf, e := LoadSshConfigFile(xname)
				if e != nil {
					log.Error("%q[%d]: %q - subconfig %q error: %v",
						name, i+1, line, xname, e)
					continue
				}
				if cf == nil {
					log.Warn("%q[%d]: %q - subconfig %q has no entries",
						name, i+1, line, xname)
					continue
				}
				if cfg == nil {
					cfg = cf
				} else {
					cfg.Append(cf)
				}
				cf = nil
			}
			continue
		}
		if strings.HasPrefix(line, "Host") {
			// Host *
			if !entry.IsNull() {
				if cfg == nil {
					cfg = new(SshConfigFile).SetName(fname)
				}
				cfg.Set(host, entry)
			}
			host = strings.TrimSpace(line[4:])
			entry = new(SshConfigFileEntry)
			continue
		}
		if leadingSpace_re.MatchString(line) {
			line = strings.TrimSpace(spaces_re.ReplaceAllString(line, sp))
			word := strings.Split(line, sp)
			tag, word := word[0], word[1:]
			entry.Set(tag, strings.Join(word, sp))
			continue
		}
		log.Error("%q[%d]: %q - bad entry", name, i+1, line)
	}
	if entry.IsNull() {
		entry = nil // let GC to do its job, if any
	} else {
		if cfg == nil {
			cfg = new(SshConfigFile).SetName(fname)
		}
		cfg.Set(host, entry)
	}
	log.Debug("Loaded %q (entries: %d)", fname, cfg.Length())
	return
}

/* EOF */
