// Doost!

package main

import (
	"github.com/alphazero/gart/syslib/errors"
)

func (cmd *Cmd) helpCmd([]string) error {
	cmd.run = helpCommand
	return nil
}

func helpCommand() error {
	return errors.NotImplemented("cmd/helpCommand")
}
