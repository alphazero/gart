// Doost!

package main

import (
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type listOption struct {
	force bool
}

func (cmd *Cmd) listCmd(args []string) error {
	var option listOption

	flags := flag.NewFlagSet("gart-list", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-listialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	cmd.option = option
	cmd.run = listCommand
	return nil
}

func listCommand() error {
	return errors.NotImplemented("cmd/listCommand")
}
