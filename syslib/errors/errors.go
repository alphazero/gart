// Doost!

package errors

import (
	"fmt"
)

/// defined generic errors /////////////////////////////////////////////////////

var (
	ErrNotImplemented = Error("function not implemented")
	ErrInvalidArg     = Error("invalid argument")
	ErrNilArg         = Error("nil input argument")
)

/// defined generic bugs ///////////////////////////////////////////////////////

var (
	// bugs are for internal use and assert checks.
	BugInvalidArg = Bug("invalid input argument")
	BugNilArg     = Bug("nil input argument")
)

/// defined generic faults /////////////////////////////////////////////////////

var (
	FaultNotImplemented = Fault("function not implemented")
)

/// function specific errors ///////////////////////////////////////////////////

type fnErrors string

func For(fname string) Errors {
	return fnErrors(fname)
}
func (fn fnErrors) InvalidArg(fs string, a ...interface{}) error {
	return Error(string(fn)+": invalid arg - "+fs, a...)
}
func (fn fnErrors) NotImplemented() error                   { return Error(string(fn) + ": not implemented") }
func (fn fnErrors) Error(fs string, a ...interface{}) error { return Error(string(fn)+": "+fs, a...) }
func (fn fnErrors) Bug(fs string, a ...interface{}) error   { return Bug(string(fn)+": "+fs, a...) }
func (fn fnErrors) Fault(fs string, a ...interface{}) error { return Fault(string(fn)+": "+fs, a...) }
func (fn fnErrors) ErrorWithCause(e error, fs string, a ...interface{}) error {
	return ErrorWithCause(e, string(fn)+": "+fs, a...)
}
func (fn fnErrors) BugWithCause(e error, fs string, a ...interface{}) error {
	return BugWithCause(e, string(fn)+": "+fs, a...)
}
func (fn fnErrors) FaultWithCause(e error, fs string, a ...interface{}) error {
	return FaultWithCause(e, string(fn)+": "+fs, a...)
}

type Errors interface {
	NotImplemented() error
	InvalidArg(fmtstr string, a ...interface{}) error
	Error(fmtstr string, a ...interface{}) error
	Bug(fmtstr string, a ...interface{}) error
	Fault(fmtstr string, a ...interface{}) error
	ErrorWithCause(e error, fmtstr string, a ...interface{}) error
	BugWithCause(e error, fmtstr string, a ...interface{}) error
	FaultWithCause(e error, fmtstr string, a ...interface{}) error
}

/// err/bug uniform formatters /////////////////////////////////////////////////

func InvalidArg(where, what, why string) error {
	return fmterr("err", "%s: invalid arg - %s is %s", where, what, why)
}

func NotImplemented(fmtstr string, a ...interface{}) error {
	return fmterr("err", fmtstr+" is not implemented", a...)
}

func Usage(fmtstr string, a ...interface{}) error {
	return fmterr("usage", fmtstr, a...)
}

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
	//	return fmt.Errorf(what+" - "+fmtstr, a...)
	return fmt.Errorf(what+": "+fmtstr, a...)
}
