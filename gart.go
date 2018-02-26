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
			onError(meta, e)
			err = e
			break
		}

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
