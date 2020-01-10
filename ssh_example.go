package main

import (
	"flag"
	"github.com/jn0/go-log"
)

var log = logging.Root

func main() {
	log.Info("started")
	defer log.Info("stopped")

	log.SetLevel(logging.DEBUG)
	log.UsePanic(true)

	use_term := false
	host := "host.example.com"
	cmd := "pwd"

	flag.BoolVar(&use_term, "term", false, "request tty")
	flag.Parse()
	if flag.NArg() > 0 {
		host = flag.Arg(0)
	}
	if flag.NArg() > 1 {
		cmd = flag.Arg(1)
	}

	context := NewContext(host, use_term)
	out := RunCommandOverSsh(context, cmd)
	log.Info("Got %q", out)

}
