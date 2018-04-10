// Doost!

package main

import (
	"fmt"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/system"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// throw some errors:

	debug.Printf("%v", system.ErrIndexExist)
	debug.Printf("%v", system.ErrIndexNotExist)
}
