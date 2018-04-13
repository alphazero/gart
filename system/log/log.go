// Doost!

package log

import (
	"fmt"
	"io"
)

var Log func(string, ...interface{})

func init() {
	Log = quietLog()
}

// REVU later TODO quiet still writes to .gart/log/<rotate>.log[.n]
func quietLog() func(string, ...interface{}) {
	return func(string, ...interface{}) {}
}

func verboseLog(w io.Writer) func(string, ...interface{}) {
	return func(fmtstr string, a ...interface{}) {
		fmt.Fprintf(w, fmtstr+"\n", a...)
	}
}

func Verbose(w io.Writer) {
	Log = verboseLog(w)
}
