// Doost!

package main

import (
	"context"

	"github.com/alphazero/gart/syslib/errors"
)

func parseHelpArgs(args []string) (Command, Option, error) {
	return helpCommand, nil, nil
}

func helpCommand(context.Context, Option) error {
	return errors.NotImplemented("cmd.helpCommand")
}
