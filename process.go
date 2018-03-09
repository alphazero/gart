// Friend

package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
)

/// general process ///////////////////////////////////////////////////////////

// exit codes
const (
	EC_OK = iota
	EC_USAGE
	EC_ERROR
	EC_INTERRUPT
	EC_FAULT
)

// gart process info
type processInfo struct {
	name    string     // process (base)name
	wd      string     // process working dir
	user    *user.User // process user
	gartDir string     // gart home
	in      io.Reader  // process input stream   REVU not sure if necessary
	out     io.Writer  // process output stream  REVU not sure if necessary
	meta    io.Writer  // process out-of-band output stream
}

func processPrepare(in io.Reader, out, meta io.Writer) (context.Context, error) {
	pi, e := getProcessInfo(in, out, meta)
	if e != nil {
		return nil, e
	}

	// setup command context & state
	var state State
	ctx := context.WithValue(context.Background(), "state", &state)
	state.pi = pi

	return ctx, cmdPrepare(&state)
}

func getState(ctx context.Context) *State {
	// binding must be present, of correct type, and non-nil
	// If not, we have a bug
	state, ok := ctx.Value("state").(*State)
	if !ok || state == nil {
		panic("bug")
	}
	return state
}

func getProcessInfo(in io.Reader, out, meta io.Writer) (processInfo, error) {
	var pi processInfo
	user, e := user.Current()
	if e != nil {
		return pi, e
	}
	pi.user = user

	if len(os.Args) < 1 {
		panic("bug -- os.Args is zerolen")
	}
	pi.name = filepath.Base(os.Args[0])

	pi.gartDir = filepath.Join(user.HomeDir, gartDir)
	pi.in = in
	pi.out = out
	pi.meta = meta

	return pi, nil
}

/// process shell /////////////////////////////////////////////////////////////

type Mode int

const (
	Fault Mode = iota
	Standalone
	Piped
)

var flags = flag.NewFlagSet("cmdline options", flag.ContinueOnError)

func main() {
	var in io.Reader

	mode, e := parseFlags(os.Args[1:])
	if e != nil {
		os.Exit(EC_USAGE)
	}
	switch mode {
	case Standalone:
		buf := bytes.NewBufferString(flags.Arg(0) + "\n")
		in = bufio.NewReader(buf)
	case Piped:
		in = os.Stdin
	default:
		panic("bug")
	}

	if e := processStream(in, os.Stdout, os.Stderr); e != nil {
		os.Exit(EC_ERROR)
	}
}

// Parse flags and also determines the run mode.
// Most of this is delegated to the actual command (file)
func parseFlags(args []string) (Mode, error) {
	flags.SetOutput(os.Stderr)
	if e := flags.Parse(args); e != nil {
		return Fault, e
	}

	if e := checkFlags(); e != nil {
		return Fault, e
	}

	return processMode(), nil
}

func processStream(in io.Reader, out, meta io.Writer) (err error) {

	var silent bool = true // REVU generic proc needs to get this from flags TODO

	/// process loop //////////////////////////////////////

	// prepare for processing.
	ctx, e := processPrepare(in, out, meta)
	if e != nil {
		onError(meta, e)
		return e
	}

	// shutdown on interrupt
	interrupt := make(chan os.Signal, 1)
	defer close(interrupt)
	go func() {
		_ = <-interrupt
		onReturnOrExit(ctx, &err, meta)
		os.Exit(EC_INTERRUPT)
	}()
	signal.Notify(interrupt, os.Interrupt, os.Kill)

	// normal shutdown of process on return
	defer onReturnOrExit(ctx, &err, meta)

	/// process loop //////////////////////////////////////

	var r = bufio.NewReader(in)
	var w = bufio.NewWriter(out)
	defer func() { w.Flush() }()
	var abort bool
	for !abort {
		line, e := r.ReadBytes('\n')
		if e == io.EOF {
			break
		}
		if e != nil {
			onError(meta, e) // REVU fail-stop
			err = e
			break
		}

		result, e, abort := process(ctx, line[:len(line)-1])
		if abort {
			break
		}
		if e != nil {
			onError(meta, e)
			continue
		}
		if !silent && result != nil {
			w.Write(result)
			w.WriteByte('\n')
		}
	}
	return
}

// REVU processShutdown (rename) is hidden here in the code and the
// name doesn't quite capture it.
func onReturnOrExit(ctx context.Context, err *error, w io.Writer) {
	signal.Reset(os.Interrupt, os.Kill)
	if e := processDone(ctx); e != nil {
		onError(w, e)
		*err = e // only meaningful on returns
	}
}

func onError(w io.Writer, e error) {
	w.Write(append([]byte(e.Error()), '\n'))
}
