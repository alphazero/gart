// Doost!

package main

import (
	"fmt"
	"time"

	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/systemic"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// throw some errors:

	debug.Printf("%v", system.ErrIndexExist)
	debug.Printf("%v", system.ErrIndexNotExist)

	fmt.Printf("%s\n", systemic.ExtTag(""))
	fmt.Printf("%s\n", systemic.ExtTag("pdf"))
	fmt.Printf("%s\n", systemic.TypeTag(system.URL.String()))
	fmt.Printf("%s\n", systemic.TypeTag("file"))
	fmt.Printf("%s\n", systemic.DayTag(time.Now()))
	fmt.Printf("%s\n", systemic.TodayTag())
}
