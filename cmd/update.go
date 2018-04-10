// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type updateOption struct {
	force bool // XXX
}

func parseUpdateArgs(args []string) (Command, Option, error) {
	var option updateOption

	flags := flag.NewFlagSet("gart-update", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-updateialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	return updateCommand, option, nil
}

func updateCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.updateCommand")

	_, ok := option0.(updateOption)
	if !ok {
		return err.InvalidArg("expecting updateOption - %v", option0)
	}
	return err.NotImplemented()
}
