package main

import (
//	"bytes"
	"flag"
	"github.com/jn0/go-log"
	"golang.org/x/crypto/ssh"
	"os/user"
)

var log = logging.Root

func main() {
	log.Info("started")
	defer log.Info("stopped")

	log.SetLevel(logging.DEBUG)
	log.UsePanic(true)

	use_term := false
	flag.BoolVar(&use_term, "term", false, "request tty")

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
			User:            sshcf.Get(host, "User", u.Username),
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(LoadPrivateKey())},
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

	if use_term {
		term := "dumb"
		modes := ssh.TerminalModes{
			ssh.ECHO: 1,
			ssh.TTY_OP_ISPEED: 19200,
			ssh.TTY_OP_OSPEED: 19200,
		}
		log.Debug("Requesting a tty (%q %v)", term, modes)
		err := sess.RequestPty(term, 80, 25, modes)
		if err != nil {
			log.Fatal("Cannot request tty: %v", err)
		}
		log.Debug("Got a tty!")
	}

	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		log.Error("SSH session (%q): %v", cmd, err)
	}
	log.Info("Got %q", string(out))

/*
	var out bytes.Buffer
	sess.Stdout = &out

	log.Info("Running %q", cmd)
	err = sess.Run(cmd)
	if err != nil {
		log.Error("SSH session (%q): %v", cmd, err)
	}
	log.Info("Got %q", out.String())
*/
}
