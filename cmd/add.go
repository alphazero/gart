// Doost!

package main

import (
	"bufio"
	"context"
	"flag"
	"io"
	"os"
	"strings"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/log"
)

type addOption struct {
	cmdOption
	text    bool
	url     bool
	tagspec string
	args    []string
	otype   system.Otype
}

// gart add -tags "tag1, tag 2, tag 3" f1.ext f2.ext ...
// gart add --strict -tags "tag1, tag 2, tag 3" f1.ext f2.ext ...
// gart add --text -tags "tag1, tag 2, tag 3" "quote 1" "quote 2" ...
// cat pithy.txt | gart add --test -tags "pithy quotes"
// find . -type f -name "*.pdf" | gart add --test -tags "pithy quotes"
func parseAddArgs(args []string) (Command, Option, error) {
	var option = addOption{
		otype: system.File,
	}

	flags := flag.NewFlagSet("gart add", flag.ExitOnError)
	option.usingVerboseFlag(flags)
	option.usingStrictFlag(flags, "add new objects only - no updates")
	flags.BoolVar(&option.text, "text", option.text, "archive text object(s) -- overrides default file type")
	flags.BoolVar(&option.url, "url", option.url, "archive url object(s) -- overrides default file type")
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
	switch {
	case option.text && option.url:
		return err.InvalidArg("flags text and url are mutually exlusive")
	case option.text:
		option.otype = system.Text
	case option.url:
		option.otype = system.URL
		return err.Error("Url objects not yet supported")
	}
	if len(option.args) == 0 {
		return addStreamedObjects(ctx, option)
	}
	return addSpecifiedObjects(ctx, option)
}

func addSpecifiedObjects(ctx context.Context, option addOption) error {
	var err = errors.For("cmd.addSpecifiedObjects")
	//	var debug = debug.For("cmd.addSpecifiedObjects")
	//	debug.Printf("options:%v\n", option)

	session, e := gart.OpenSession(ctx, gart.Add)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	defer func() {
		session.Close()
		log.Log("session - close")
	}()

	log.Log("session - begin")

	var tags = parseTags(option.tagspec)
	for _, spec := range option.args {
		if len(spec) == 0 {
			continue
		}
		card, added, e := session.AddObject(option.strict, option.otype, spec, tags...)
		if e != nil {
			if index.IsObjectExistErr(e) {
				log.Log("%s exists - %q", e.(index.Error).Oid.Fingerprint(), spec)
				continue
			}
			return e
		}
		log.Log("%s (added: %t) %q", card.Oid().Fingerprint(), added, spec)
	}
	return nil
}

// TODO signal handling
func addStreamedObjects(ctx context.Context, option addOption) error {
	var err = errors.For("cmd.addStreamedObjects")
	var debug = debug.For("cmd.addStreamedObjects")

	session, e := gart.OpenSession(ctx, gart.Add)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	defer session.Close()
	debug.Printf("using session: %v", session)

	var tags = parseTags(option.tagspec)
	var r = bufio.NewReader(os.Stdin)
	for {
		line, e := r.ReadBytes('\n')
		if e != nil {
			break
		}
		spec := string(line[:len(line)-1])
		if len(spec) == 0 {
			continue
		}
		card, added, e := session.AddObject(option.strict, option.otype, spec, tags...)
		if e != nil {
			if index.IsObjectExistErr(e) {
				log.Log("%s exists - %q", e.(index.Error).Oid.Fingerprint(), spec)
				continue
			}
			return e
		}
		log.Log("%s (added: %t) %q", card.Oid().Fingerprint(), added, spec)
	}
	if e == io.EOF {
		return nil
	}
	return e
}
