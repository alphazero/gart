// Doost!

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/log"
	"github.com/alphazero/gart/system/systemic"
)

type findOption struct {
	cmdOption
	incTags, exTags   string
	incTypes, exTypes string
	incExts, exExts   string
	date              string // TODO ex: -d [-/+]mar-19-2018
	digest            bool
}

func parseFindArgs(args []string) (Command, Option, error) {
	var option findOption

	option.flags = flag.NewFlagSet("gart find", flag.ExitOnError)
	option.usingVerboseFlag("verbose cmd op")
	option.flags.BoolVar(&option.digest, "digest", option.digest,
		"print single line digest of matching object")

	option.flags.StringVar(&option.incTypes, "types", option.incTypes,
		"objects of type (csv list of {file, text, url, uri})")
	option.flags.StringVar(&option.exTypes, "x-types", option.exTypes,
		"exclude objects of type (csv list of {file, text, url, uri})")

	option.flags.StringVar(&option.incExts, "exts", option.incExts,
		"file objects with extention (csv list)")
	option.flags.StringVar(&option.exExts, "x-exts", option.exExts,
		"exclude file objects with extension (csv list)")

	option.flags.StringVar(&option.incTags, "tags", option.incTags,
		"objects with tags (csv list)")
	option.flags.StringVar(&option.exTags, "x-tags", option.exTags,
		"exclude objects with tags (csv list)")

	// default gart find w/ no tags returns all objects
	if len(args) > 1 {
		option.flags.Parse(args[1:])
	}

	return findCommand, option, nil
}

func findCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.findCommand")
	var debug = debug.For("cmd.findCommand")
	debug.Printf("context:%v\n", ctx)

	option, ok := option0.(findOption)
	if !ok {
		return err.InvalidArg("expecting findOption - %v", option)
	}
	debug.Printf("options:%v\n", option)

	/// gart session ////////////////////////////////////////////////

	var ctxChild, cancel = context.WithCancel(ctx)
	session, e := gart.OpenSession(ctxChild, gart.Find)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	defer func() {
		session.Close(false) // REVU don't like commit as Session.Close arg ..
		log.Log("session - close")
	}()

	/// find query spec /////////////////////////////////////////////

	var qbuilder = gart.NewQuery()

	// user tags
	qbuilder.IncludeTags(parseCsv(option.incTags)...)
	qbuilder.ExcludeTags(parseCsv(option.exTags)...)

	// systemic flags
	for _, s := range parseCsv(option.incTypes) {
		qbuilder.IncludeTags(systemic.TypeTag(s))
	}
	for _, s := range parseCsv(option.exTypes) {
		qbuilder.ExcludeTags(systemic.TypeTag(s))
	}
	for _, s := range parseCsv(option.incExts) {
		qbuilder.IncludeTags(systemic.ExtTag(s))
	}
	for _, s := range parseCsv(option.exExts) {
		qbuilder.ExcludeTags(systemic.ExtTag(s))
	}

	/// async exec //////////////////////////////////////////////////

	oc, ec := session.AsyncExec(qbuilder.Build())
	e = nil
loop:
	for {
		select {
		case obj := <-oc:
			card, ok := obj.(index.Card)
			if !ok {
				cancel()
			}
			if obj == nil {
				break loop // done
			}
			// TODO tigheten up emit options in flags
			if option.isVerbose() {
				card.Print(os.Stdout) // TODO verbose flag for card
			} else {
				var digest string
				switch card.Type() {
				case system.Text:
					digest = card.(index.TextCard).Text()
				case system.File:
					fcard := card.(index.FileCard)
					paths := fcard.Paths()
					if len(paths) > 1 {
						digest = fmt.Sprintf("(dup:%d) ", len(paths)-1)
					}
					digest += fmt.Sprintf("%s", paths[0])
				}
				fmt.Printf("oid:%s version:%d [%s] %s\n",
					card.Oid().Fingerprint(), card.Version(), card.Type(), digest)
			}
		case e = <-ec:
			if e != nil {
				break loop
			}
		}
	}
	cancel() // REVU is this necessary ?

	return e
}
