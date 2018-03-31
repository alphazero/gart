// Doost!

package main

import (
	"fmt"

	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	fmt.Printf("%s\n", errors.ErrInvalidArg)
	fmt.Printf("%s\n", system.BugInvalidOidBytesData)
	fmt.Printf("%s\n", errors.BugInvalidArg)
	fmt.Printf("%s\n", errors.NotImplemented("gart/syslib/errors/adhoc/main"))
	fmt.Printf("%s\n", errors.Usage("unknown option %s", "foo"))

	e := fmt.Errorf("not really an error")
	msg := "Salaam Samad Sultan of LOVE"
	fmt.Printf("%s\n", errors.ErrorWithCause(e, "adhoc.main: errors check - msg %s", msg))
	fmt.Printf("%s\n", errors.InvalidArg("adhoc.main", "foo", "<= 0"))
}
