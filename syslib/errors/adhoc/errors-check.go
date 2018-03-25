// Doost!

package main

import (
	"fmt"

	"github.com/alphazero/gart/syslib/errors"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	fmt.Printf("%s\n", errors.ErrInvalidArg)
	fmt.Printf("%s\n", errors.BugInvalidOidBytesData)
	fmt.Printf("%s\n", errors.BugInvalidArg)

	e := fmt.Errorf("not really an error")
	msg := "Salaam Samad Sultan of LOVE"
	fmt.Printf("%s\n", errors.ErrorWithCause(e, "adhoc.main: errors check - msg %s", msg))
}
