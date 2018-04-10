// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type tagOption struct {
	force bool // XXX
}

func parseTagArgs(args []string) (Command, Option, error) {
	var option tagOption

	flags := flag.NewFlagSet("gart-tag", flag.ExitOnError)
	flags.BoolVar(&option.force, "force", option.force, "force re-tagialization of repo")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	return tagCommand, option, nil
}

func tagCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.tagCommand")

	_, ok := option0.(tagOption)
	if !ok {
		return err.InvalidArg("expecting tagOption - %v", option0)
	}
	return err.NotImplemented()
}
