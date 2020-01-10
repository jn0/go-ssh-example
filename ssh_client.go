package main

import (
	"golang.org/x/crypto/ssh"
	"strings"
)

/*============================================================================*/

func NewSshClientConfig(context map[string]string) *ssh.ClientConfig {
	host, ok := context["host"]
	if !ok {
		log.Fatal("Context %v has no \"host\" entry", context)
	}

	hkey := FindHostKeyByContext(context)
	if hkey == nil {
		log.Fatal("No known host key for %+q", host)
	}

	return &ssh.ClientConfig{
		User:            context["user"],
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(LoadPrivateKey())},
		HostKeyCallback: ssh.FixedHostKey(hkey),
	}
}

func NewSshClient(context map[string]string) *ssh.Client {
	host, ok := context["host"]
	if !ok {
		log.Fatal("Context %v has no \"host\" entry", context)
	}

	port, ok := context["port"]
	if !ok {
		port = "22"
	}

	clnt, err := ssh.Dial("tcp", host+":"+port, NewSshClientConfig(context))
	if err != nil {
		log.Fatal("SSH client: %v", err)
	}
	return clnt
}

func RunCommandOverSsh(context map[string]string, command string, args ...string) string {
	sshc := NewSshClient(context)
	defer sshc.Close()

	sess, err := sshc.NewSession()
	if err != nil {
		log.Fatal("SSH client session: %v", err)
	}
	defer sess.Close()

	term, ok := context["term"]
	if ok && term != "" {
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
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

	cmd := command
	if len(args) > 0 {
		cmd += " " + strings.Join(args, " ")
	}

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

	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		log.Error("SSH session (%q): %v", cmd, err)
	}

	return string(out)
}

/* EOF */
