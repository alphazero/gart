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
	home string // gart home
	pwd  string // process working directory
}

/// command processing ////////////////////////////////////////////////////////

// pre:
func processPrepare() (context.Context, error) {
	// setup command context & state
	var state State
	ctx := context.WithValue(context.Background(), "state", &state)

	pwd, e := os.Getwd()
	if e != nil {
		return ctx, e
	}
	state.pwd = pwd

	return ctx, nil
}

// command gart-add
// Returns output, error if any, and abort
func process(ctx context.Context, b []byte) (output []byte, err error, abort bool) {

	state := ctx.Value("state").(*State)
	if state == nil {
		panic("bug")
	}

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
	//	return md, nil
	output = []byte(fmt.Sprintf("%x %s", md, fds.Path))
	return
}

// post:
func processDone(ctxt context.Context) error {
	return nil
}
