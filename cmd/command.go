// Doost!

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/log"
)

var _ = system.Debug

type Command func(context.Context, Option) error
type Option interface {
	isVerbose() bool
	isStrict() bool
}
type cmdOption struct {
	verbose bool
	strict  bool
}

func (p *cmdOption) usingVerboseFlag(fs *flag.FlagSet) {
	fs.BoolVar(&(*p).verbose, "verbose", p.verbose, "verbose emits op log to stderr")
}
func (p *cmdOption) usingStrictFlag(fs *flag.FlagSet, info string) {
	fs.BoolVar(&(*p).strict, "strict", p.strict, info)
}
func (v cmdOption) isVerbose() bool { return v.verbose }
func (v cmdOption) isStrict() bool  { return v.strict }

func parseArgs(args []string) (Command, Option, error) {
	var cname string

	if len(args) == 1 {
		cname = "help"
	} else {
		if args[1][0] == '-' {
			return nil, nil, ErrUsage
		}
		cname = args[1]
	}

	switch cname {
	case "help":
		return parseHelpArgs(nil)
	case "version":
		return parseVersionArgs(nil)
	case "list":
		return parseListArgs(args[1:])
	case "init":
		return parseInitArgs(args[1:])
	case "add":
		return parseAddArgs(args[1:])
	case "delete":
		return parseDeleteArgs(args[1:])
	case "update":
		return parseUpdateArgs(args[1:])
	case "find":
		return parseFindArgs(args[1:])
	case "tag":
		return parseTagArgs(args[1:])
	}

	debug.Printf("unknown command - args: %q", args)
	return nil, nil, ErrUsage
}

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	command, option, e := parseArgs(os.Args)
	switch e {
	case nil:
	case ErrUsage:
		exitOnUsage()
	case ErrInterrupt:
		exitOnInterrupt()
	default:
		exitOnError(e)
	}

	if option.isVerbose() {
		log.Verbose(os.Stderr)
	}

	var ctx = context.Background()
	if e := command(ctx, option); e != nil {
		exitOnError(e)
	}

	os.Exit(0)
}

/// exit handling //////////////////////////////////////////////////////////////

var (
	ErrUsage     = errors.Error("usage")
	ErrInterrupt = errors.Error("interrupted")
)

// exit codes
const (
	EC_OK = iota
	EC_USAGE
	EC_ERROR
	EC_INTERRUPT
	EC_FAULT
)

func exitOnUsage() {
	fmt.Fprintf(os.Stderr, "%v\n", errors.NotImplemented("cmd/usage"))
	os.Exit(EC_USAGE)
}

func exitOnInterrupt() {
	fmt.Fprintf(os.Stderr, "%v\n", errors.NotImplemented("cmd/onInterrupt"))
	os.Exit(EC_USAGE)
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%v\n", e)
	os.Exit(EC_ERROR)
}
