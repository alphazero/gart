// Doost!

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/log"
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
		option.oid = strings.Split(option.oid, ".")[0]
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

	// use provided oid, or optionally compute from file if using path
	oid := option.oid
	if option.usepath {
		path, e := filepath.Abs(option.path)
		if e != nil {
			return err.ErrorWithCause(e, "unexpected error on filepath.Abs")
		}
		md, e := digest.SumFile(path)
		if e != nil {
			return err.ErrorWithCause(e, "error computing digest for file")
		}
		p, e := system.NewOid(md)
		if e != nil {
			return err.ErrorWithCause(e, "unexpected error creating Oid for file digest ")
		}
		oid = strings.Split(p.Fingerprint(), ".")[0]
	}
	cards, e := gart.FindCard(oid)
	if e != nil {
		return e
	}

	switch len(cards) {
	case 0:
		return errors.Error("no cards found for %s", oid)
	case 1:
	default:
		msg := fmt.Sprintf("Ambiguous oid pattern - found %d cards for %s",
			len(cards), oid)

		log.Log(msg)
		for _, card := range cards {
			log.Log("-> %s %s", card.Type(), card.Oid().Fingerprint())
		}
		return err.Error(msg)
	}

	var card = cards[0]
	switch option.isVerbose() {
	case true:
		card.Print(os.Stdout) // TODO add verbose flag to Card.Print
	default:
		card.Print(os.Stdout) // TODO minimal info emit for card
	}

	return nil
}
