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
	Error Mode = iota
	Standalone
	Piped
)

func main() {
	var in io.Reader
	//	var fidx int = 1

	// piped mode or strictly cmdline opts?
	//	if len(os.Args) > 1 && os.Args[1][0] != '-' {
	//		buf := bytes.NewBufferString(os.Args[1] + "\n")
	//		in = bufio.NewReader(buf)
	//		fidx = 2
	//	}

	mode, e := parseFlags(os.Args[1:])
	if e != nil {
		exit.OnError(e)
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
		exit.OnError(e)
	}
}

var flags = flag.NewFlagSet("options", flag.ContinueOnError)

func parseFlags(args []string) (Mode, error) {
	flags.SetOutput(os.Stderr)
	if e := flags.Parse(args); e != nil {
		return Error, e
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
			meta.Write([]byte(e.Error()))
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
