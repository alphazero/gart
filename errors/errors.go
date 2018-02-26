package errors

import (
	"fmt"
)

var Usage = fmt.Errorf("usage error. use -h flag for correct usage")
var IllegalArgument = fmt.Errorf("illegal argument")
var IrregualFile = fmt.Errorf("not a regular file")
