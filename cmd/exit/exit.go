package exit

import (
	"fmt"
	"os"
	"path/filepath"
)

// exit codes
const (
	EC_OK = iota
	EC_ERROR
	EC_USAGE
	EC_INTERRUPT
)

// Exits the process with EC_OK code. Nothing emitted.
func Ok() {
	os.Exit(EC_OK)
}

// Exits the process with EC_ERROR code. Error message is emitted to Stderr
func OnError(e error, detail ...interface{}) {
	var fmtstr = "err - %v"
	var args []interface{}
	args = append(args, e)
	for _, v := range detail {
		args = append(args, v)
		fmtstr += " %v"
	}
	emit(fmtstr, args...)
	os.Exit(EC_ERROR)
}

// Exits the process with EC_USAGE code. Usage message is emitted to Stderr
func OnUsage(usage string) {
	emit("usage: %s %s", filepath.Base(os.Args[0]), usage)
	os.Exit(EC_USAGE)
}

func OnInterrupt(sig os.Signal) {
	emit("interrupt: %d", sig)
	os.Exit(EC_INTERRUPT)
}

func emit(fmtstr string, args ...interface{}) (int, error) {
	return fmt.Fprintf(os.Stderr, fmtstr+"\n", args...)
}
