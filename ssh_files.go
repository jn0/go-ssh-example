package main

import (
	"io/ioutil"
	"os"
	"os/user"
	fpth "path"
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

func findSshFile(username, filename string) (path string, err error) {
	var u *user.User
	if username != "" {
		u, err = user.Lookup(username)
		if err != nil {
			log.Error("User %q lookup error: %v", username, err)
			return
		}
	} else {
		u, err = user.Current()
		if err != nil {
			log.Error("Current user lookup error: %v", err)
			return
		}
	}
	path = fpth.Join(u.HomeDir, filename)
	_, err = os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		log.Warn("No file %q", path)
	}
	return
}

func FindHostKeyFile(username string) string {
	path, err := findSshFile(username, DefaultSshKnownHosts)
	if err != nil {
		log.Fatal("Cannot find known hosts %q for user %q in %q: %v",
			DefaultSshKnownHosts, username, path, err)
	}
	return path
}

func FindSshPubKeyFile(username string) string {
	path, err := findSshFile(username, DefaultSshKeyFile+DefaultSshKeyFilePubSuffix)
	if err != nil {
		log.Fatal("Cannot find pub key %q for user %q in %q: %v",
			DefaultSshKeyFile+DefaultSshKeyFilePubSuffix,
			username, path, err)
	}
	return path
}

func FindSshPvtKeyFile(username string) string {
	path, err := findSshFile(username, DefaultSshKeyFile)
	if err != nil {
		log.Fatal("Cannot find key %q for user %q in %q: %v",
			DefaultSshKeyFile, username, path, err)
	}
	return path
}

func FindSshConfigFile(username string) string {
	path, err := findSshFile(username, DefaultSshConfigFile)
	if err != nil {
		log.Fatal("Cannot find config %q for user %q in %q: %v",
			DefaultSshConfigFile, username, path, err)
	}
	return path
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

	fname := name
	if name == "" || name == DefaultSshConfigFile {
		fname = FindSshConfigFile("")
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
		if strings.HasPrefix(line, "Host") {
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
	log.Info("Loaded %q", fname)
	return
}

/* EOF */
