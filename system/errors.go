// Doost!

package system

import (
	"fmt"
)

/// defined errors /////////////////////////////////////////////////////////////

var (
	ErrInvalidArg = Error("invalid argument")
)

/// defined bugs ///////////////////////////////////////////////////////////////

var (
	BugInvalidOidBytesData = Bug("invalid oid bytes data")
	BugInvalidArg          = Bug("invalid input argument")
	BugNilArg              = Bug("nil input argument")
)

/// defined faults /////////////////////////////////////////////////////////////

/// err/bug uniform formatters /////////////////////////////////////////////////

func Error(fmtstr string, a ...interface{}) error {
	return fmterr("err", fmtstr, a...)
}

func Bug(fmtstr string, a ...interface{}) error {
	return fmterr("bug", fmtstr, a...)
}

func Fault(fmtstr string, a ...interface{}) error {
	return fmterr("fault", fmtstr, a...)
}

func ErrorWithCause(e error, fmtstr string, a ...interface{}) error {
	return fmterr("err", fmtstr+" - cause: %v", append(a, e)...)
}

func BugWithCause(e error, fmtstr string, a ...interface{}) error {
	return fmterr("bug", fmtstr+" - cause: %v", append(a, e)...)
}

func FaultWithCause(e error, fmtstr string, a ...interface{}) error {
	return fmterr("fault", fmtstr+" - cause: %v", append(a, e)...)
}

func fmterr(what, fmtstr string, a ...interface{}) error {
	return fmt.Errorf(what+" - "+fmtstr, a...)
}
