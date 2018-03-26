// Doost!

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

// to try:
// using system fixtures
// - create a Wahl tagmap
// - open Wahl tagmap in OpUpdate mode
//	- save using mmap
//		- multiple Sets then Compress then Sync
//
// - load Wahl in OpQuery mode
//		- query 'keys' that are set
//
// - tagmap manager
//		- AND 2 or more tagmaps and get 'keyset'

var option = struct {
	op   string
	tags string
}{}
var ops = []string{"c", "r", "w", "q"}

func init() {
	flag.StringVar(&option.op, "op", option.op, "c:create, r:read, w:write, q:query")
	flag.StringVar(&option.tags, "tags", option.tags, "csv list in \" \"s ")
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "exit on error: %v\n", e)
	os.Exit(1)
}

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// parse flags and verify option
	flag.Parse()

	if option.op == "" {
		exitOnError(errors.Usage("flag -op must be specified"))
	}
	option.op = strings.ToLower(option.op)
	for _, op := range ops {
		if op == option.op {
			goto op_verified
		}
	}
	exitOnError(errors.Usage("invalid op:%q", option.op))

op_verified:
	option.tags = strings.TrimSuffix(option.tags, ",")
	var tagnames = strings.Split(option.tags, ",")
	for i, s := range tagnames {
		tag := strings.Trim(s, " ")
		if tag == "" {
			exitOnError(errors.Usage("option -tags has zero-len tagname: %q", tagnames))
		}
		tagnames[i] = tag
	}
	if len(tagnames) == 0 {
		exitOnError(errors.Usage("option -tags must be non-empty"))
	}

	var e error
	switch option.op {
	case "c":
		e = createTagmaps(system.RepoPath, tagnames...)
	case "r":
		e = readTagmap(system.RepoPath, tagnames[0])
	case "w":
		e = writeTagmap(system.RepoPath, tagnames[0])
	case "q":
		e = queryTagmaps(system.RepoPath, tagnames...)
	default:
		exitOnError(errors.Bug("verified op is not known: %q", option.op))
	}

	if e != nil {
		exitOnError(errors.Bug("op: %q -  %v", option.op, e))
	}

	os.Exit(0)
}

/// op prototypes //////////////////////////////////////////////////////////////

func createTagmaps(repoDir string, tags ...string) error {
	return errors.NotImplemented("createTagmaps")
}

func readTagmap(repoDir string, tag string) error {
	return errors.NotImplemented("readTagmap")
}

func writeTagmap(repoDir string, tag string) error {
	return errors.NotImplemented("writeTagmap")
}

func queryTagmaps(repoDir string, tags ...string) error {
	return errors.NotImplemented("queryTagmaps")
}
