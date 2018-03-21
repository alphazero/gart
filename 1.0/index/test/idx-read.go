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
	testReadMode1()
}

func testReadMode1() {
	var e error
	idx, e := index.OpenIdxFile(garthome, index.IdxRead)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("openned idx file: %s\n", idx.Filename())
	fmt.Printf("  objects: %d\n", idx.Size())
	fmt.Printf("debug:\n%v\n", idx)

	// OK - close here
	if e = idx.Close(); e != nil {
		exitOnError(e)
	}
	fmt.Printf("closed idx file: %s\n", idx.Filename())

	// error on close here
	e = idx.Close()
	if e != index.ErrIdxClosed {
		exitOnError(fmt.Errorf("bug - expected ErrIdxClosed error on close here: e: %v", e))
	}
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
