// Friend

package main

import (
	"bufio"
	"bytes"
	"context"
	"github.com/alphazero/gart/cmd/exit"
	"io"
	"os"
	"os/signal"
)

func main() {
	var in io.Reader

	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		buf := bytes.NewBufferString(os.Args[1] + "\n")
		in = bufio.NewReader(buf)
	} else {
		in = os.Stdin
	}

	e := processStream(in, os.Stdout, os.Stderr)
	if e != nil {
		exit.OnError(e)
	}
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
