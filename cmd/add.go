// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart/syslib/errors"
)

type addOption struct {
	text string
	file string
	tags string
}

func parseAddArgs(args []string) (Command, Option, error) {
	var option addOption
	flags := flag.NewFlagSet("gart-add", flag.ExitOnError)
	flags.StringVar(&option.file, "file", option.file, "path of file object to archive")
	flags.StringVar(&option.text, "text", option.text, "content of text object to archive")
	flags.StringVar(&option.tags, "tags", option.tags, "csv list of tags to apply to object")
	if len(args) > 2 {
		flags.Parse(args[1:])
	}

	return addCommand, option, nil
}

func addCommand(ctx context.Context, option Option) error {
	var err = errors.For("cmd.addCommand")

	option, ok := option.(addOption)
	if !ok {
		return err.InvalidArg("expecting addOption - %v", option)
	}
	return err.NotImplemented()
}
