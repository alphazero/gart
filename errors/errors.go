package errors

import (
	"fmt"
)

var Usage = fmt.Errorf("usage error. use -h flag for correct usage")
var IllegalArgument = fmt.Errorf("illegal argument")
var IrregularFile = fmt.Errorf("not a regular file")

func Errorf(e error, a ...interface{}) error {
	var fmtstr = "err - %v"
	var args []interface{}
	args = append(args, e)
	for _, v := range a {
		args = append(args, v)
		fmtstr += " %v"
	}
	return fmt.Errorf(fmtstr, args...)
}
