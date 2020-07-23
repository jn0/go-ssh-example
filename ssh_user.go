package main

import (
	"os/user"
)

func NewContext(host string, use_term bool) map[string]string {
	u, err := user.Current()
	if err != nil {
		log.Fatal("No current user: %v", err)
	}
	log.Debug("Running as %q (%s)", u.Username, u.Name)

	sshcfg := NewSshConfig()

	context := map[string]string{
		"host": host,
		"port": sshcfg.GetValue(host, "Port", "22"),
		"user": sshcfg.GetValue(host, "User", u.Username),
	}
	if use_term {
		context["term"] = "pty"
	}

	return context
}

/* EOF */
