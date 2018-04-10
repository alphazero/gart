// Doost!

package debug

import (
	"fmt"
	"io"
	"runtime"
	"strings"
)

// Set by system.init(), either /dev/null or stderr (if system.Debug is true)
var Writer io.Writer

type Printer interface {
	Printf(string, ...interface{})
}

type fnPrinter string

func For(fname string) Printer { return fnPrinter(fname) }
func (v fnPrinter) Printf(fmtstr string, a ...interface{}) {
	printf(3, string(v)+": "+fmtstr, a...)
}

func Printf(fmtstr string, a ...interface{}) {
	printf(2, fmtstr, a...)
}

func printf(level int, fmtstr string, a ...interface{}) {
	if Writer == nil {
		return
	}
	var prefix = "debug: "
	_, file, line, ok := runtime.Caller(level)
	if ok {
		gpp := "gart/"
		cpx := strings.LastIndex(file, gpp) + len(gpp)
		prefix = fmt.Sprintf("debug [%s:%d]: ", file[cpx:], line)
	}
	fmt.Fprintf(Writer, prefix+fmtstr+"\n", a...)
}
