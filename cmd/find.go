// Doost!

package main

import (
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type findOption struct {
	force bool
}

func (cmd *Cmd) findCmd(args []string) error {
	var option findOption

	flags := flag.NewFlagSet("gart-find", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-findialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	cmd.option = option
	cmd.run = findCommand
	return nil
}

func findCommand() error {
	return errors.NotImplemented("cmd/findCommand")
}
