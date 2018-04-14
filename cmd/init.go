// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system/log"
)

type initOption struct {
	cmdOption
	force bool
}

func parseInitArgs(args []string) (Command, Option, error) {
	var option initOption

	option.flags = flag.NewFlagSet("gart-init", flag.ExitOnError)
	option.usingVerboseFlag0()
	option.flags.BoolVar(&option.force, "force", option.force,
		"force re-initialization of repo")

	if len(args) > 1 {
		option.flags.Parse(args[1:])
	}

	return initCommand, option, nil
}

func initCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.initCommand")

	option, ok := option0.(initOption)
	if !ok {
		return err.InvalidArg("expecting initOption - %v", option0)
	}

	ok, e := gart.InitRepo(option.force)
	if ok {
		log.Log("initialized repo")
	}
	return e
}
