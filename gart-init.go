// Doost

package main

import (
	"context"
	"fmt"
)

/// flags and processing mode /////////////////////////////////////////////////

var option = struct {
	force  bool
	silent bool
}{}

func init() {
	flags.BoolVar(&option.force, "force", option.force, "force re-initialization of gart repo")
	flags.BoolVar(&option.silent, "silent", option.silent, "silent re-init (with --force only)")
}

// Check mandatory flags, etc.
func checkFlags() error {
	return nil // TODO if silent is true, the force must also be true
}

// Each process determines the run mode per its cmd-line options pattern
func processMode() Mode {
	return Standalone
}

/// command specific state ////////////////////////////////////////////////////

// struct encapsulates mutable and immutable process values.
type State struct {
	pi processInfo
}

/// command processing ////////////////////////////////////////////////////////

// pre:
func cmdPrepare(state *State) error { return nil }

// command gart-init REVU build flags?
// Returns output, error if any, and abort
func process(ctx context.Context, b []byte) (output []byte, err error, abort bool) {

	state := getState(ctx)

	if e := initGartRepo(state.pi, option.force, option.silent); e != nil {
		fmt.Fprintln(state.pi.meta, e)
		err = fmt.Errorf("fatal - existing gart repo. run 'gart-init --force'")
		return
	}
	output = []byte(fmt.Sprintf("initialized gart repo at %q\n", state.pi.gartDir))
	return
}

// post:
func processDone(ctxt context.Context) error {
	// TODO close .gart/index/tags.idx in APPEND mode.
	// REVU unlock it ?

	return nil
}
