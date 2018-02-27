// Friend

package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"github.com/alphazero/gart/cmd/exit"
	"io"
	"os"
	"os/signal"
)

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
		os.Exit(exit.EC_USAGE)
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
		os.Exit(exit.EC_ERROR)
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

	/// process loop //////////////////////////////////////

	// REVU error here is fundamentally fatal
	// prepare for processing.
	ctx, e := processPrepare()
	if e != nil {
		onError(meta, e)
		return e
	}

	// shutdown on interrupt
	interrupt := make(chan os.Signal, 1)
	defer close(interrupt)
	go func() {
		sig := <-interrupt
		onReturnOrExit(ctx, &err, meta)
		exit.OnInterrupt(sig)
	}()
	signal.Notify(interrupt, os.Interrupt, os.Kill)

	// normal shutdown of process on return
	defer onReturnOrExit(ctx, &err, meta)

	/// process loop //////////////////////////////////////

	var r = bufio.NewReader(in)
	var w = bufio.NewWriter(out)
	defer func() { w.Flush() }()
	for {
		line, e := r.ReadBytes('\n')
		if e == io.EOF {
			break
		}
		if e != nil {
			onError(meta, e) // REVU fail-stop
			err = e
			break
		}

		// REVU two flavors of errors are required: fail-stop and item specific
		// error. For example, if find . is piped to gart-add, the stream may be
		// a mix of directories and files. We don't want to stop in the middle of
		// the stream with a cryptic "not a regular file".
		//
		// in general, each distinct process has d distinct logging policy and distinct
		// set of fail-stop and (effectively) warnings.
		// this means errors package needs to distinguish between FatalError and Error.
		//
		// (REVU or even simpler, have process return
		// func process(context.Context, []byte) ([]byte, error, bool).
		// res, e, fatal := process(ctx, line[:..])
		// if fatal { onError(e); err = e; break }
		// if e != nil { onError(e); continue }
		result, e := process(ctx, line[:len(line)-1])
		if e != nil {
			onError(meta, e)
			err = e
			break
		}
		if result != nil {
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
