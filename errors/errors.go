package errors

import (
	"fmt"
)

var IllegalArgument = fmt.Errorf("illegal argument")
var IrregualFile = fmt.Errorf("not a regular file")
