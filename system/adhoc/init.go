// Doost!

package main

import (
	"fmt"
	"github.com/alphazero/gart/system"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// throw some errors:

	system.Debugf("%v", system.ErrIndexExist)
	system.Debugf("%v", system.ErrIndexNotExist)
}
