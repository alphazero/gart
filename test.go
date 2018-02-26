// Doost

package main

import (
	"context"
	"fmt"
)

var option = struct {
	test string
}{
	test: "default",
}

func init() {
	flags.StringVar(&option.test, "t", option.test, "test option")
}

// NOTE each cmd has to determine the run mode.
func processMode() Mode {
	if flags.NArg() == 0 {
		return Piped
	}
	return Standalone
}

type State struct {
	items int
}

// pre:
func processPrepare() (context.Context, error) {
	// check flags

	// setup command context & state
	var state State
	ctx := context.WithValue(context.Background(), "state", &state)

	return ctx, nil
}

// post:
func processDone(ctxt context.Context) error {
	return nil
}

// command:
func process(ctx context.Context, b []byte) ([]byte, error) {

	state := ctx.Value("state").(*State)
	s := []byte(fmt.Sprintf("%03d - %q - [%d] // %s", state.items, string(b), len(b), option.test))
	state.items++

	return s, nil
}
