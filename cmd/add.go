// Doost!

package main

import (
	"bufio"
	"context"
	"flag"
	"io"
	"os"

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

	option.flags = flag.NewFlagSet("gart add", flag.ExitOnError)
	option.usingVerboseFlag0()
	option.usingStrictFlag("add new objects only - no updates")
	option.flags.BoolVar(&option.text, "text", option.text,
		"archive text object(s) -- overrides default file type")
	option.flags.BoolVar(&option.url, "url", option.url,
		"archive url object(s) -- overrides default file type")
	option.flags.StringVar(&option.tagspec, "tags", option.tagspec,
		"required - csv list of tags to apply to object")

	var debug = debug.For("cmd.parseAddArgs")

	if len(args) < 2 {
		debug.Printf("no flags specified")
		return nil, option, ErrUsage
	}

	option.flags.Parse(args[1:])
	if option.tagspec == "" {
		debug.Printf("tags flag is required")
		return nil, option, ErrUsage
	}
	option.args = option.flags.Args()

	return addCommand, option, nil
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

	session, e := gart.OpenSession(ctx, gart.Add)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	log.Log("session - begin")

	var tags = parseTags(option.tagspec)
	for _, spec := range option.args {
		if len(spec) == 0 {
			continue
		}
		if e = interruptibleAdd(ctx, session, option.strict, option.otype, spec, tags...); e != nil {
			log.Error(e.Error())
			break
		}
	}
	var commit = e == nil // do not commit on any error
	if ec := session.Close(commit); ec != nil {
		panic(err.Fault("on session close - %v", ec))
	} else {
		log.Log("session - close")
	}
	return e
}

func addStreamedObjects(ctx context.Context, option addOption) error {
	var err = errors.For("cmd.addStreamedObjects")

	session, e := gart.OpenSession(ctx, gart.Add)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	log.Log("session - begin")

	var tags = parseTags(option.tagspec)
	var r = bufio.NewReader(os.Stdin)
	var line []byte
	var n int
	for {
		line, e = r.ReadBytes('\n')
		if e != nil {
			break
		}
		n++
		spec := string(line[:len(line)-1])
		if len(spec) == 0 {
			continue
		}
		if e = interruptibleAdd(ctx, session, option.strict, option.otype, spec, tags...); e != nil {
			log.Error(e.Error())
			break
		}
	}

	var commit = e == nil || e == io.EOF // do not commit on any other error
	if ec := session.Close(commit); ec != nil {
		panic(err.Fault("on session close - %v", ec))
	}
	log.Log("session - close - %d items processed", n)

	if e == io.EOF {
		if n == 0 {
			return err.Error("no items were read from stdin - eof")
		}
		return nil
	}

	return e
}

func interruptibleAdd(ctx context.Context, session gart.Session, strict bool, typ system.Otype, spec string, tags ...string) error {
	select {
	case <-ctx.Done():
		return ErrInterrupt
	default:
		card, added, e := session.AddObject(strict, typ, spec, tags...)
		if e != nil {
			if index.IsObjectExistErr(e) {
				log.Log("%s exists - %q", e.(index.Error).Oid.Fingerprint(), spec)
				return nil
			}
			return e
		}
		log.Log("%s (added: %t) %q", card.Oid().Fingerprint(), added, spec)
	}
	return nil
}
