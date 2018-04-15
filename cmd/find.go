// Doost!

package main

import (
	"context"
	"flag"
	"os"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/log"
	"github.com/alphazero/gart/system/systemic"
)

// gart find -tags "a, b, c" -type file -ext "pdf"
type findOption struct {
	cmdOption
	tagspec string // REVU this should be a type that support flag.Value
	//	exclude string
	otype  system.Otype
	ext    string
	date   string // TODO ex: -d [-/+]mar-19-2018
	digest bool
}

func parseFindArgs(args []string) (Command, Option, error) {
	//	var deftyp = system.File
	var option = findOption{
		otype: system.File,
		ext:   "-",
	}

	option.flags = flag.NewFlagSet("gart find", flag.ExitOnError)
	option.usingVerboseFlag("verbose cmd op")
	option.flags.Var(&option.otype, "type",
		"filter by object type")
	option.flags.StringVar(&option.ext, "ext", option.ext,
		"filter by file extension (file type only)")
	option.flags.BoolVar(&option.digest, "digest", option.digest,
		"print single line digest of matching object")
	option.flags.StringVar(&option.tagspec, "tags", option.tagspec,
		"with tags tags (csv list)")
	//	option.flags.StringVar(&option.tagspec, "exclude", option.tagspec,
	//		"excluding tags (csv list)")

	// default gart find w/ no tags returns all objects
	if len(args) > 1 {
		option.flags.Parse(args[1:])
	}

	return findCommand, option, nil
}

func findCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.findCommand")
	var debug = debug.For("cmd.findCommand")

	option, ok := option0.(findOption)
	if !ok {
		return err.InvalidArg("expecting findOption - %v", option)
	}
	debug.Printf("options:\n%v\n", option)

	var systemics []string
	if option.otype != 0 {
		systemics = append(systemics, systemic.TypeTag(option.otype.String()))
		if option.otype == system.File && option.ext != "-" {
			systemics = append(systemics, systemic.ExtTag(option.ext))
		}
	}

	session, e := gart.OpenSession(ctx, gart.Add)
	if e != nil {
		return err.Error("could not open session - %v", e)
	}
	defer func() {
		session.Close()
		log.Log("session - close")
	}()

	cards, e := session.Select()
	if e != nil {
		return e
	}

	for _, card := range cards {
		// TODO tigheten up emit options in flags
		card.Print(os.Stdout)
	}

	return nil
}
