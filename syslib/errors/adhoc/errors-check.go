// Doost!

package main

import (
	"fmt"

	"github.com/alphazero/gart/syslib/errors"
<<<<<<< HEAD
	"github.com/alphazero/gart/system"
=======
>>>>>>> gart-2.0-bitmap
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	fmt.Printf("%s\n", errors.ErrInvalidArg)
<<<<<<< HEAD
	fmt.Printf("%s\n", system.BugInvalidOidBytesData)
	fmt.Printf("%s\n", errors.BugInvalidArg)
	fmt.Printf("%s\n", errors.NotImplemented("gart/syslib/errors/adhoc/main"))
	fmt.Printf("%s\n", errors.Usage("unknown option %s", "foo"))
=======
	fmt.Printf("%s\n", errors.BugInvalidOidBytesData)
	fmt.Printf("%s\n", errors.BugInvalidArg)
>>>>>>> gart-2.0-bitmap

	e := fmt.Errorf("not really an error")
	msg := "Salaam Samad Sultan of LOVE"
	fmt.Printf("%s\n", errors.ErrorWithCause(e, "adhoc.main: errors check - msg %s", msg))
}
