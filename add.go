// Doost

package main

import (
	"context"
	"fmt"
	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/file"
	"os"
)

/// flags and processing mode /////////////////////////////////////////////////

var option = struct {
	test string
}{
	test: "default",
}

func init() {
	flags.StringVar(&option.test, "t", option.test, "test option")
}

// Check mandatory flags, etc.
func checkFlags() error {
	return nil
}

// Each process determines the run mode per its cmd-lien options pattern
func processMode() Mode {
	if flags.NArg() == 0 {
		return Piped
	}
	return Standalone
}

/// command specific state ////////////////////////////////////////////////////

type State struct {
	home  string // gart home
	pwd   string // process working directory
	items int
}

/// command processing ////////////////////////////////////////////////////////

// pre:
func processPrepare() (context.Context, error) {
	// setup command context & state
	var state State
	ctx := context.WithValue(context.Background(), "state", &state)

	// REVU this is not necessary ?
	pwd, e := os.Getwd()
	if e != nil {
		return ctx, e
	}
	state.pwd = pwd

	// TODO open .gart/index/tags.idx in APPEND mode.
	// TODO open .gart/tags/tagsdef in RW mode.
	// REVU lock it ?

	return ctx, nil
}

// REVU TODO this should be in common
func getState(ctx context.Context) *State {
	// binding must be present, of correct type, and non-nil
	// If not, we have a bug
	state, ok := ctx.Value("state").(*State)
	if !ok || state == nil {
		panic("bug")
	}
	return state
}

// command gart-add
// Returns output, error if any, and abort
func process(ctx context.Context, b []byte) (output []byte, err error, abort bool) {

	state := getState(ctx)
	defer func() { state.items++ }()

	fds, e := file.GetDetails(string(b))
	if e != nil {
		return nil, e, false // REVU don't abort - next file may be ok
	}
	//	fmt.Printf("%v\n", fds)

	// REVU TODO digest code needs to use RDONLY open ..
	md, e := digest.Compute(fds.Path)
	if e != nil {
		return nil, e, false // TODO REVU
	}

	// REVU
	//		- update .gart/paths (if required)
	//		  REVU check state to see if this has already been done
	//		- create tags bitmap REVU default tags (e.g. ext) + cmdline options
	//        if --tags "...." are spec'd in options, update tags/tagsdef
	//			REVU this includes updating frequency count of the tags
	// 		- create card file
	//		  REVU card may already exist.
	//		  REVU if --no-dups is specified, return nil error but emit msg to stderr
	// 		- append to index/TAGS
	//		  REVU state should have this file open in APPEND mode already.

	// XXX temporary
	output = []byte(fmt.Sprintf("%x %s", md, fds.Path))
	return
}

// post:
func processDone(ctxt context.Context) error {
	// TODO close .gart/index/tags.idx in APPEND mode.
	// REVU unlock it ?

	return nil
}
