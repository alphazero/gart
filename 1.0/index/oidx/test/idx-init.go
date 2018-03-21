// Doost!

package main

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/index/oidx"
)

const garthome = "/Users/alphazero/.gart"

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// just load the idxfile.go for init() checks
	e := oidx.CreateIndex(garthome)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("idx file created\n")
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
