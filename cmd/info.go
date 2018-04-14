// Doost!

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alphazero/gart"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system/log"
)

type infoOption struct {
	cmdOption
	oid string
}

func parseInfoArgs(args []string) (Command, Option, error) {
	var option infoOption

	option.flags = flag.NewFlagSet("gart info", flag.ExitOnError)
	option.usingVerboseFlag("print all details")
	option.flags.StringVar(&option.oid, "oid", option.oid,
		"required - oid of object")

	if len(args) > 1 {
		option.flags.Parse(args[1:])
		if option.oid == "" {
			return nil, option, ErrUsage
		}
	} else {
		return nil, option, ErrUsage
	}

	return infoCommand, option, nil
}

func infoCommand(ctx context.Context, option0 Option) error {
	var err = errors.For("cmd.infoCommand")
	var debug = debug.For("cmd.infoCommand")

	option, ok := option0.(infoOption)
	if !ok {
		return err.InvalidArg("expecting infoOption - %v", option0)
	}

	debug.Printf("oid:%q", option.oid)
	cards, e := gart.FindCard(option.oid)
	if e != nil {
		return e
	}

	switch len(cards) {
	case 0:
		return errors.Error("no cards found for %s", option.oid)
	case 1:
	default:
		msg := fmt.Sprintf("Ambiguous oid pattern - found %d cards for %s",
			len(cards), option.oid)

		log.Log(msg)
		for _, card := range cards {
			log.Log("-> %s %s", card.Type(), card.Oid().Fingerprint())
		}
		return err.Error(msg)
	}

	var card = cards[0]
	switch option.isVerbose() {
	case true:
		debug.Printf("todo - verbose info emit for card\n%s\n", card.Oid())
		card.Print(os.Stdout)
	default:
		debug.Printf("todo - minimal info emit for card\n%s\n", card.Oid())
		card.Print(os.Stdout)
	}

	return nil
}
