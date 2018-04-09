// Doost!

package main

import (
	//	"flag"
	"fmt"
	"os"

	"github.com/alphazero/gart/syslib/errors"
	//	"github.com/alphazero/gart/system"
)

type Cmd struct {
	name string
	//	args []string
	//	flags  *flag.FlagSet
	option interface{}
	run    func() error
}

func (cmd *Cmd) parseArgs(args []string) error {
	if len(args) == 1 {
		cmd.name = "help"
	} else {
		if args[1][0] == '-' {
			return ErrUsage
		}
		cmd.name = args[1]
	}

	switch cmd.name {
	case "help":
		return cmd.helpCmd(nil)
	case "version":
		return cmd.versionCmd(nil)
	case "list":
		return cmd.listCmd(args[1:])
	case "init":
		return cmd.initCmd(args[1:])
	case "add":
		return cmd.addCmd(args[1:])
	case "delete":
		return cmd.deleteCmd(args[1:])
	case "update":
		return cmd.updateCmd(args[1:])
	case "find":
		return cmd.findCmd(args[1:])
	case "tag":
		return cmd.tagCmd(args[1:])
	}

	return errors.NotImplemented("cmd.parseArgs")
}

func usage() {}

var ErrUsage = errors.Error("usage")

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	var cmd Cmd
	if e := cmd.parseArgs(os.Args); e != nil {
		exitOnError(e)
	}
	if e := cmd.run(); e != nil {
		exitOnError(e)
	}

	os.Exit(0)
}

// exit codes
const (
	EC_OK = iota
	EC_USAGE
	EC_ERROR
	EC_INTERRUPT
	EC_FAULT
)

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%v\n", e)
	os.Exit(EC_ERROR)
}
