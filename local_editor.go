package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var envEditorNames = []string{"VISUAL", "EDITOR"}

func _edit(editor, file string) error {
	log.Debug("Trying %q on %q", editor, file)
	cmd := exec.Command(editor, file)
	if !FileExists(cmd.Path) {
		return errors.New(fmt.Sprintf("No %q", cmd.Path))
	}
	log.Debug("%q %v", cmd.Path, cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(fmt.Sprintf("Error running %q: %v", cmd.Path, err))
	}
	return nil
}

const MailcapEditor = "edit"
const TheEditor = "vi"

func Edit(name string) {
	log.Debug("Trying to edit %q", name)
	if name == "" {
		return
	}
	for _, ev := range envEditorNames {
		ed, ok := os.LookupEnv(ev)
		if !ok || ed == "" {
			continue
		}
		err := _edit(ed, name)
		if err != nil {
			log.Warn("%q=%q: %v", ev, ed, err)
			continue
		}
		return
	}
	if _edit(MailcapEditor, name) == nil {
		return
	}
	if _edit(TheEditor, name) == nil { // last resort
		return
	}
	log.Fatal("No editor found")
}
