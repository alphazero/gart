// Doost!

package main

import (
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type initOption struct {
	force bool
}

func (cmd *Cmd) initCmd(args []string) error {
	var option initOption

	flags := flag.NewFlagSet("gart-init", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-initialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	cmd.option = option
	cmd.run = initCommand
	return nil
}

func initCommand() error {
	return errors.NotImplemented("cmd/initCommand")
}
