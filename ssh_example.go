package main

import (
	"github.com/jn0/go-log"
	"golang.org/x/crypto/ssh"
	"os/user"
	"flag"
	"bytes"
)

var log = logging.Root

func main() {
	log.Info("started")
	defer log.Info("stopped")

	log.SetLevel(logging.DEBUG)
	log.UsePanic(true)
	flag.Parse()

	u, err := user.Current()
	if err != nil {
		log.Fatal("No current user: %v", err)
	}
	log.Info("Running as %q (%s)", u.Username, u.Name)

	sshcf, err := LoadSshConfigFile(DefaultSshConfigFile)
	if err != nil {
		log.Fatal("SSH config file: %v", err)
	}
	log.Info("SSH config file %q has %d host entries", sshcf.Name(), sshcf.Len())
	// log.Say("Config:\n%s\n", sshcf)

	host := "host.example.com"
	cmd := "pwd"
	if flag.NArg() > 0 {
		host = flag.Arg(0)
	}
	if flag.NArg() > 1 {
		cmd = flag.Arg(1)
	}

	sshc, err := ssh.Dial(
		"tcp",
		host+":"+sshcf.Get(host, "Port", "22"),
		&ssh.ClientConfig{
			User: sshcf.Get(host, "User", u.Username),
			Auth: []ssh.AuthMethod{ssh.PublicKeys(LoadPrivateKey())},
			HostKeyCallback: ssh.FixedHostKey(FindHostKey(host)),
		},
	)
	if err != nil {
		log.Fatal("SSH client: %v", err)
	}
	defer sshc.Close()

	sess, err := sshc.NewSession()
	if err != nil {
		log.Fatal("SSH client session: %v", err)
	}
	defer sess.Close()

	var out bytes.Buffer
	sess.Stdout = &out

	log.Info("Running %q", cmd)
	err = sess.Run(cmd)
	if err != nil {
		log.Fatal("SSH session (%q): %v", cmd, err)
	}
	log.Info("Got %q", out.String())
}
