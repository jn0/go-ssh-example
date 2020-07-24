package main

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh/agent"
)

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
