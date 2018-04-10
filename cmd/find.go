// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type findOption struct {
	force bool // XXX
}

func parseFindArgs(args []string) (Command, Option, error) {
	var option findOption

	flags := flag.NewFlagSet("gart-find", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-findialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	return findCommand, option, nil
}

func findCommand(ctx context.Context, option Option) error {
	var err = errors.For("cmd.findCommand")

	option, ok := option.(findOption)
	if !ok {
		return err.InvalidArg("expecting findOption - %v", option)
	}
	return err.NotImplemented()
}
