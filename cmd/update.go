// Doost!

package main

import (
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type updateOption struct {
	force bool
}

func (cmd *Cmd) updateCmd(args []string) error {
	var option updateOption

	flags := flag.NewFlagSet("gart-update", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-updateialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	cmd.option = option
	cmd.run = updateCommand
	return nil
}

func updateCommand() error {
	return errors.NotImplemented("cmd/updateCommand")
}
