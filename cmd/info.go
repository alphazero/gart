// Doost!

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	//	"strings"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

type infoOption struct {
	cmdOption
	oid     string
	path    string
	usepath bool
}

func parseInfoArgs(args []string) (Command, Option, error) {
	var option infoOption

	option.flags = flag.NewFlagSet("gart info", flag.ExitOnError)
	option.usingVerboseFlag("emit oids in case of multiple results")
	option.flags.BoolVar(&option.usepath, "file", option.usepath, "check file instead of oid")

	// parse flags, expecting filename or oid as remaining arg
	if len(args) > 1 {
		option.flags.Parse(args[1:])
	} else {
		return nil, option, ErrUsage
	}
	if len(option.flags.Args()) < 1 {
		return nil, option, ErrUsage
	}

	switch option.usepath {
	case true:
		option.path = option.flags.Args()[0]
	default:
		option.oid = option.flags.Args()[0]
		if option.oid == "" {
			return nil, option, ErrUsage
		}
	}

	return infoCommand, option, nil
}

func infoCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.infoCommand")

	option, ok := option0.(infoOption)
	if !ok {
		return err.InvalidArg("expecting infoOption - %v", option0)
	}

	// use provided oid fingerprint, or optionally compute from file if using path
	fingerprint := option.oid
	if option.usepath {
		oid, e := getOidForFile(option.path)
		if e != nil {
			return err.ErrorWithCause(e, "unexpected error creating Oid for file digest ")
		}
		fingerprint = oid.Fingerprint()
	}
	cards, e := gart.FindCard(fingerprint)
	if e != nil {
		return e
	}

	switch len(cards) {
	case 0:
		return errors.Error("no cards found for %s", fingerprint)
	default:
		for _, card := range cards {
			switch option.isVerbose() {
			case true:
				card.Print(os.Stdout)
			default:
				fmt.Fprintf(os.Stdout, "%s\n", card.Info())
			}
		}
	}
	return nil
}

func getOidForFile(path string) (*system.Oid, error) {
	var err = errors.For("cmd.getOidForFile")
	path, e := filepath.Abs(path)
	if e != nil {
		return nil, err.ErrorWithCause(e, "unexpected error on filepath.Abs")
	}
	md, e := digest.SumFile(path)
	if e != nil {
		return nil, err.ErrorWithCause(e, "error computing digest for file")
	}
	return system.NewOid(md)
}
