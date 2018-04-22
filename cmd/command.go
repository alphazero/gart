// Doost!

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system/log"
)

/// command generalization /////////////////////////////////////////////////////

// general command func
type Command func(context.Context, Option) error

// base command options
type Option interface {
	flagSet() *flag.FlagSet
	isVerbose() bool
	isStrict() bool
}

// cmdOption supports Option interface and provides struct base for command
// specific options.
type cmdOption struct {
	flags   *flag.FlagSet
	verbose bool
	strict  bool
}

func (p *cmdOption) setFlagSet(fs *flag.FlagSet) { p.flags = fs }

func (p *cmdOption) usingVerboseFlag0() {
	p.usingVerboseFlag("verbose emits op log to stderr")
}

func (p *cmdOption) usingVerboseFlag(info string) {
	p.flags.BoolVar(&(*p).verbose, "verbose", p.verbose, info)
}

func (p *cmdOption) usingStrictFlag(info string) {
	p.flags.BoolVar(&(*p).strict, "strict", p.strict, info)
}

func (v cmdOption) flagSet() *flag.FlagSet { return v.flags }
func (v cmdOption) isVerbose() bool        { return v.verbose }
func (v cmdOption) isStrict() bool         { return v.strict }

/// uniform command-line arg pre-processing ////////////////////////////////////

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
	case "info":
		return parseInfoArgs(args[1:])
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

/// command-line process ///////////////////////////////////////////////////////

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	command, option, e := parseArgs(os.Args)
	switch e {
	case nil:
	case ErrUsage:
		var fs *flag.FlagSet
		if option != nil {
			fs = option.flagSet()
		}
		exitOnUsage(fs)
	case ErrInterrupt:
		exitOnInterrupt()
	default:
		exitOnError(e)
	}

	// help, etc. have no flags
	if option != nil && option.isVerbose() {
		log.Verbose(os.Stderr)
	}

	var ctx = interruptibleContext(context.Background())
	e = command(ctx, option)
	switch e {
	case nil:
	case ErrInterrupt:
		exitOnInterrupt()
	default:
		exitOnError(e)
	}
	os.Exit(0)
}

/// exit handling //////////////////////////////////////////////////////////////

func interruptibleContext(parent context.Context) context.Context {
	ctx, cancel := context.WithCancel(parent)
	go func(cancel func()) {
		var ch = make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, os.Kill)
		<-ch
		cancel()
	}(cancel)
	return ctx
}

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

func exitOnUsage(flags *flag.FlagSet) {
	if flags != nil {
		flags.SetOutput(os.Stderr)
		flags.Usage()
	} else {
		fmt.Fprintf(os.Stderr, "%v\n", errors.NotImplemented("gart top-level usage"))
	}
	os.Exit(EC_USAGE)
}

func exitOnInterrupt() {
	log.Log("interrupted")
	os.Exit(EC_INTERRUPT)
}

func exitOnError(e error) {
	log.Error("%v", e)
	os.Exit(EC_ERROR)
}
