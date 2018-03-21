// Doost!

package main

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/index"
)

const garthome = "/Users/alphazero/.gart"

func main() {
	fmt.Printf("Salaam!\n")

	// just load the idxfile.go for init() checks
	e := index.CreateIdxFile(garthome)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("idx file created\n")
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
