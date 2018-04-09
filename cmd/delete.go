// Doost!

package main

import (
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type deleteOption struct {
	force bool
}

func (cmd *Cmd) deleteCmd(args []string) error {
	var option deleteOption

	flags := flag.NewFlagSet("gart-delete", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-deleteialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	cmd.option = option
	cmd.run = deleteCommand
	return nil
}

func deleteCommand() error {
	return errors.NotImplemented("cmd/deleteCommand")
}
