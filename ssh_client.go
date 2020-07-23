package main

import (
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

/*============================================================================*/

const SSH_AUTH_SOCK = "SSH_AUTH_SOCK"

func newSshAgentClient() agent.ExtendedAgent {
	socket := os.Getenv(SSH_AUTH_SOCK)
	conn, err := net.Dial("unix", socket)
	if err != nil {
		log.Error("Cannot use %s=%q: %v", SSH_AUTH_SOCK, socket, err)
		return nil
	}
	return agent.NewClient(conn)
}

func authMethods() []ssh.AuthMethod {
	var auth []ssh.AuthMethod
	agn := newSshAgentClient()
	if agn != nil {
		log.Debug("Using agent via %q", os.Getenv(SSH_AUTH_SOCK))
		auth = append(auth, ssh.PublicKeysCallback(agn.Signers))
	}
	pk := LoadPrivateKey()
	if pk != nil {
		log.Debug("Using private key")
		auth = append(auth, ssh.PublicKeys(LoadPrivateKey()))
	}
	return auth
}

func hostKeyMethod(context map[string]string) ssh.HostKeyCallback {
	hkey := FindHostKeyByContext(context)
	if hkey == nil {
		log.Error("No known host key for %+q", context["host"])
	} else {
		return ssh.FixedHostKey(hkey)
	}
	return ssh.InsecureIgnoreHostKey()
}

func NewSshClientConfig(context map[string]string) *ssh.ClientConfig {
	if _, ok := context["host"]; !ok {
		log.Fatal("Context %v has no \"host\" entry", context)
	}

	return &ssh.ClientConfig{
		User:            context["user"],
		Auth:            authMethods(),
		BannerCallback:  ssh.BannerDisplayStderr(),
		HostKeyCallback: hostKeyMethod(context),
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

	clnt, err := ssh.Dial("tcp", net.JoinHostPort(host, port), NewSshClientConfig(context))
	if err != nil {
		log.Fatal("SSH client[%v:%v]: %v", host, port, err)
	}
	return clnt
}

func RunCommandOverSsh(context map[string]string, command string, args ...string) (string, error) {
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
		log.Debug("SSH session (%q): %v", cmd, err)
	}

	return string(out), err
}

/* EOF */
