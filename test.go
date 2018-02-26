// Doost

package main

import (
	"context"
	"fmt"
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
	items int
}

/// command processing ////////////////////////////////////////////////////////

// pre:
func processPrepare() (context.Context, error) {
	// setup command context & state
	var state State
	ctx := context.WithValue(context.Background(), "state", &state)

	return ctx, nil
}

// command:
func process(ctx context.Context, b []byte) ([]byte, error) {

	state := ctx.Value("state").(*State)
	s := []byte(fmt.Sprintf("%03d - %q - [%d] // %s", state.items, string(b), len(b), option.test))
	state.items++

	return s, nil
}

// post:
func processDone(ctxt context.Context) error {
	return nil
}
