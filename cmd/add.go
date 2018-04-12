// Doost!

package main

import (
	"context"
	"flag"
	"strings"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

type addOption struct {
	strict  bool
	stream  bool
	text    bool
	file    bool
	tagspec string
	args    []string
	otype   system.Otype
}

// gart add --file -tags "tag1, tag 2, tag 3" f1.ext f2.ext ...
// gart add --strict --file -tags "tag1, tag 2, tag 3" f1.ext f2.ext ...
// gart add --text -tags "tag1, tag 2, tag 3" "quote 1" "quote 2" ...
// cat pithy.txt | gart add --test -tags "pithy quotes"
// find . -type f -name "*.pdf" | gart add --test -tags "pithy quotes"
func parseAddArgs(args []string) (Command, Option, error) {
	var option = addOption{
		file:  true,
		otype: system.File,
	}

	flags := flag.NewFlagSet("gart add", flag.ExitOnError)
	flags.BoolVar(&option.strict, "strict", option.strict, "if set add will not updated existing objects")
	flags.BoolVar(&option.file, "file", option.file, "archive file object(s) - default type")
	flags.BoolVar(&option.text, "text", option.text, "archive text object(s)")
	flags.StringVar(&option.tagspec, "tags", option.tagspec, "required - csv list of tags to apply to object")
	if len(args) < 1 {
		return nil, nil, ErrUsage
	}
	flags.Parse(args[1:])
	if option.tagspec == "" {
		debug.Printf("cmd.ParseAddArgs: tags flag not provided")
		return nil, nil, ErrUsage
	}
	option.args = flags.Args()

	return addCommand, option, nil
}

func parseTags(tagspec string) []string {
	var tags []string
	for _, s := range strings.Split(tagspec, ",") {
		s = strings.Trim(s, " ")
		s = strings.ToLower(s)
		if s == "" {
			continue // ignore invalid ,, in spec
		}
		tags = append(tags, s)
	}
	return tags
}

// TODO context shutdown handler to close session.
func addCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.addCommand")

	option, ok := option0.(addOption)
	if !ok {
		return err.InvalidArg("expecting addOption - %v", option0)
	}

	// text, file, url flags are mutually exclusive
	if option.text {
		option.otype = system.Text
	}
	if len(option.args) == 0 {
		return addStreamedObjects(ctx, option)
	}
	return addSpecifiedObjects(ctx, option)
}

func addSpecifiedObjects(ctx context.Context, option addOption) error {
	var err = errors.For("cmd.addSpecifiedObjects")
	var debug = debug.For("cmd.addSpecifiedObjects")

	debug.Printf("options:%v\n", option)

	session, e := gart.OpenSession(ctx, gart.Add)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	defer session.Close()
	debug.Printf("using session: %v", session)

	tags := parseTags(option.tagspec)
	for _, s := range option.args {
		debug.Printf("add %s %s ", option.otype, s)
		card, added, e := session.AddObject(option.strict, option.otype, s, tags...)
		if e != nil {
			debug.Printf("err - adding %s %q - %s", option.otype, s, e)
			return e
		}
		debug.Printf("%s (added: %t)", card.Oid().Fingerprint(), added)
	}
	return nil
}

func addStreamedObjects(ctx context.Context, option addOption) error {
	var err = errors.For("cmd.addStreamedObjects")
	var debug = debug.For("cmd.addStreamedObjects")

	session, e := gart.OpenSession(ctx, gart.Add)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	defer session.Close()
	debug.Printf("using session: %v", session)

	// TODO read stdin until closed

	return err.NotImplemented()
}
