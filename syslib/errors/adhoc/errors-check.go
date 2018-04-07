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

	fmt.Println()
	convenience()
}

func convenience() {
	var errs = errors.For("adhoc/main.convenience")

	fmt.Printf("%s\n", errs.NotImplemented())
	fmt.Printf("%s\n", errs.InvalidArg("foo is %t", false))
	e := errs.Error("this is a test - v:%d d:%d c:%d", 1, 2, 3)
	e2 := errs.ErrorWithCause(e, "test with cause - v:%d", 123)
	fmt.Printf("%s\n", e)
	fmt.Printf("%s\n", e2)
}
