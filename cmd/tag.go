// Doost!

package main

import (
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type tagOption struct {
	force bool
}

func (cmd *Cmd) tagCmd(args []string) error {
	var option tagOption

	flags := flag.NewFlagSet("gart-tag", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-tagialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	cmd.option = option
	cmd.run = tagCommand
	return nil
}

func tagCommand() error {
	return errors.NotImplemented("cmd/tagCommand")
}
