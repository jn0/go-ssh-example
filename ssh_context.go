package main

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Context struct {
	Id           int
	User         string
	Gecos        string
	Host         string
	Port         string
	ForwardAgent bool
	UseTty       bool
	Config       *SshConfig
	Ssh          struct {
		Agent        agent.ExtendedAgent
		ClientConfig *ssh.ClientConfig
		Client       *ssh.Client
		session      *ssh.Session
	}
}

func (context *Context) Run(command string, args ...string) (out string, err error) {
	context.Connect()
	defer context.Close()

	cmd := command
	if len(args) > 0 {
		cmd += " " + strings.Join(args, " ")
	}

	/*
		var out bytes.Buffer
		sess.Stdout = &out

		log.Info("[%d] Running %q", context.Id, cmd)
		err = sess.Run(cmd)
		if err != nil {
			log.Error("[%d] SSH session (%q): %v", context.Id, cmd, err)
		}
		log.Info("[%d] Got %q", context.Id, out.String())
	*/

	data, err := context.Ssh.session.CombinedOutput(cmd)
	if err != nil {
		log.Debug("[%d] SSH session (%q): %v", context.Id, cmd, err)
	}
	out = string(data)

	return
}

func (context *Context) Validate() {
	if context.Host == "" {
		log.Fatal("[%d] Context %+v has no \"Host\" entry", context.Id, context)
	}
}

func (context *Context) Close() {
	if context.Ssh.session != nil {
		context.Ssh.session.Close()
		context.Ssh.session = nil
	}
	if context.Ssh.Client != nil {
		context.Ssh.Client.Close()
		context.Ssh.Client = nil
	}
}

func (context *Context) endpoint() string {
	return net.JoinHostPort(context.Host, context.Port)
}

func (context *Context) requestPty() {
	log.Debug("[%d] Requesting a tty", context.Id)
	err := context.Ssh.session.RequestPty(
		"pty", 80, 25,
		ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 19200,
			ssh.TTY_OP_OSPEED: 19200,
		})
	if err != nil {
		log.Fatal("[%d] Cannot request tty: %v", context.Id, err)
	}
	log.Debug("[%d] Got a tty!", context.Id)
}

func (context *Context) Connect() {
	context.Validate()

	clnt, err := ssh.Dial("tcp", context.endpoint(), context.Ssh.ClientConfig)
	if err != nil {
		log.Fatal("[%d] SSH client[%s]: %v", context.Id, context.endpoint(), err)
	}
	context.Ssh.Client = clnt

	if context.ForwardAgent {
		if context.Ssh.Agent == nil {
			log.Fatal("[%d] No agent.", context.Id)
		}
		err := agent.ForwardToAgent(context.Ssh.Client, context.Ssh.Agent)
		if err != nil {
			log.Fatal("[%d] SetupForwardKeyring: %v", context.Id, err)
		}
		log.Debug("[%d] ForwardAgent: yes", context.Id)
	}

	context.Ssh.session, err = context.Ssh.Client.NewSession()
	if err != nil {
		log.Fatal("[%d] SSH client session: %v", context.Id, err)
	}

	if context.UseTty {
		context.requestPty()
	}

	if context.ForwardAgent {
		err = agent.RequestAgentForwarding(context.Ssh.session)
		if err != nil {
			log.Fatal("[%d] agent.ForwardToRemote: %v", context.Id, err)
		}
		log.Debug("[%d] ForwardAgent: yes", context.Id)
	}
}

func (context *Context) hostKeyMethod() ssh.HostKeyCallback {
	hkey := context.findHostKey()
	if hkey == nil {
		log.Error("[%d] No known host key for %+q", context.Id, context.Host)
		return ssh.InsecureIgnoreHostKey()
	}
	return ssh.FixedHostKey(hkey)
}

func (context *Context) findHostKey() ssh.PublicKey {
	context.Validate()

	path := FindHostKeyFile("")
	log.Debug("[%d] Host keys from %q", context.Id, path)

	var tries []string

	tries = append(tries, context.Host)
	tries = append(tries, fmt.Sprintf("[%s]:%s", context.Host, context.Port))

	for _, pattern := range tries {
		r := FindHostKey(path, pattern)
		if r != nil {
			return r
		}
	}
	log.Warn("[%d] No key for host %q", context.Id, context.Host)
	return nil
}

func (context *Context) authMethods() []ssh.AuthMethod {
	var auth []ssh.AuthMethod
	if context.Ssh.Agent != nil {
		log.Debug("[%d] Using agent via %q", context.Id, os.Getenv(SSH_AUTH_SOCK))
		auth = append(auth, ssh.PublicKeysCallback(context.Ssh.Agent.Signers))
	}
	pk := LoadPrivateKey()
	if pk != nil {
		log.Debug("[%d] Using private key", context.Id)
		auth = append(auth, ssh.PublicKeys(LoadPrivateKey()))
	}
	return auth
}

func NewContext(id int, host string, use_term bool) *Context {
	u, err := user.Current()
	if err != nil {
		log.Fatal("[%d] No current user: %v", id, err)
	}
	log.Debug("[%d] Running as %q (%s)", id, u.Username, u.Name)
	cf := NewSshConfig()
	cx := Context{
		Id:           id,
		User:         u.Username,
		Gecos:        u.Name,
		Host:         host,
		Port:         cf.GetValue(host, "Port", "22"),
		UseTty:       use_term,
		ForwardAgent: cf.GetValue(host, "ForwardAgent", "no") == "yes",
		Config:       cf,
	}
	cx.Ssh.Agent = newSshAgentClient()
	cx.Ssh.ClientConfig = &ssh.ClientConfig{
		User:            cx.User,
		Auth:            cx.authMethods(),
		BannerCallback:  ssh.BannerDisplayStderr(),
		HostKeyCallback: cx.hostKeyMethod(),
	}
	return &cx
}

/* EOF */
