// Doost!

package main

import (
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type addOption struct {
	text string
	file string
	tags string
}

func (cmd *Cmd) addCmd(args []string) error {
	var option addOption
	flags := flag.NewFlagSet("gart-add", flag.ExitOnError)
	flags.StringVar(&option.file, "file", option.file, "path of file object to archive")
	flags.StringVar(&option.text, "text", option.text, "content of text object to archive")
	flags.StringVar(&option.tags, "tags", option.tags, "csv list of tags to apply to object")
	if len(args) > 2 {
		flags.Parse(args[1:])
	}

	cmd.option = option
	cmd.run = addCommand
	return nil
}

func addCommand() error {
	return errors.NotImplemented("cmd/addCommand")
}
