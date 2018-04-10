// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type deleteOption struct {
	force bool
}

func parseDeleteArgs(args []string) (Command, Option, error) {
	var option deleteOption

	flags := flag.NewFlagSet("gart-delete", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-deleteialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	return deleteCommand, option, nil
}

func deleteCommand(ctx context.Context, option Option) error {
	var err = errors.For("cmd.deleteCommand")

	option, ok := option.(deleteOption)
	if !ok {
		return err.InvalidArg("expecting deleteOption - %v", option)
	}
	return err.NotImplemented()
}
