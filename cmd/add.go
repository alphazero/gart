// Doost!

package main

import (
	"context"
	"flag"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
)

type addOption struct {
	stream bool
	text   bool
	file   bool
	tags   string
	args   []string
}

// gart add --file -tags "tag1, tag 2, tag 3" f1.ext f2.ext ...
// gart add --text -tags "tag1, tag 2, tag 3" "quote 1" "quote 2" ...
// cat pithy.txt | gart add --test -tags "pithy quotes"
// find . -type f -name "*.pdf" | gart add --test -tags "pithy quotes"
func parseAddArgs(args []string) (Command, Option, error) {
	var option addOption
	flags := flag.NewFlagSet("gart add", flag.ExitOnError)
	flags.BoolVar(&option.file, "file", option.file, "path of file object to archive")
	flags.BoolVar(&option.text, "text", option.text, "content of text object to archive")
	flags.StringVar(&option.tags, "tags", option.tags, "csv list of tags to apply to object")
	if len(args) < 1 {
		return nil, nil, ErrUsage
	}
	flags.Parse(args[1:])
	option.args = flags.Args()

	return addCommand, option, nil
}

func addCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.addCommand")

	option, ok := option0.(addOption)
	if !ok {
		return err.InvalidArg("expecting addOption - %v", option0)
	}

	if len(option.args) > 0 {
		return addStreamedObjects(ctx, option)
	}
	return addSpecifiedObjects(ctx, option)
}

func addSpecifiedObjects(ctx context.Context, option addOption) error {
	var err = errors.For("cmd.addSpecifiedObjects")
	var debug = debug.For("cmd.addSpecifiedObjects")
	var session = gart.OpenSession(ctx)

	debug.Printf("session: %v", session)
	session.Close(ctx)
	return err.NotImplemented()
}

func addStreamedObjects(ctx context.Context, option addOption) error {
	var err = errors.For("cmd.addStreamedObjects")
	var debug = debug.For("cmd.addStreamedObjects")
	var session = gart.OpenSession(ctx)

	debug.Printf("session: %v", session)
	session.Close(ctx)
	return err.NotImplemented()
}
