// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type listOption struct {
	cmdOption
	force bool // XXX
}

func parseListArgs(args []string) (Command, Option, error) {
	var option listOption

	flags := flag.NewFlagSet("gart-list", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-listialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	return listCommand, option, nil
}

func listCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.listCommand")

	_, ok := option0.(listOption)
	if !ok {
		return err.InvalidArg("expecting listOption - %v", option0)
	}
	return err.NotImplemented()
}
