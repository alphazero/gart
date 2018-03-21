// Doost!

package main

import (
	"fmt"

	"github.com/alphazero/gart/system"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	fmt.Printf("%s\n", system.ErrInvalidArg)
	fmt.Printf("%s\n", system.BugInvalidOidBytesData)
	fmt.Printf("%s\n", system.BugInvalidArg)

	e := fmt.Errorf("not really an error")
	msg := "Salaam Samad Sultan of LOVE"
	fmt.Printf("%s\n", system.ErrorWithCause(e, "adhoc.main: errors check - msg %s", msg))
}
