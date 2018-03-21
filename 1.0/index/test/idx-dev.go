// Doost!

package main

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/unixtime"
)

const file = "/Users/alphazero/Code/go/src/gart/process.go"
const file1 = "/Users/alphazero/Code/go/src/gart/gart.go"
const file2 = "/Users/alphazero/Code/go/src/gart/gart-add.go"

// REVU idxfile has no way to know if the inputs are valid!
//      for example, if below are all same file it will happily add 3 records.
var files = []string{file, file1, file2}

const garthome = "/Users/alphazero/.gart"

func main() {
	fmt.Printf("Salaam!\n")
	testWriteMode1()
}

// create a card and then add it to idx
func testWriteMode1() {
	idx, e := index.OpenIdxFile(garthome, index.IdxUpdate)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("openned idx file: %s\n", idx.Filename())
	fmt.Printf("  objects: %d\n", idx.Size())
	fmt.Printf("debug:\n%v\n", idx)

	println("------------")
	for _, file := range files {
		// get oid, tags and systemics for the update
		oid, e := index.ObjectId(file)
		if e != nil {
			exitOnError(e)
		}
		var tags = bitmap.NewCompressed([]byte{0x7f, 0x81, 0x02})
		var systemics = bitmap.NewCompressed([]byte{0x7f})
		var date = unixtime.Now()

		key, e := idx.Add(oid, tags, systemics, date)
		if e != nil {
			exitOnError(e)
		}
		fmt.Printf("added: key: %d\n", key)
	}
	fmt.Printf("debug:\n%v\n", idx)

	println("------------")
	// Sync changes
	ok, e := idx.Sync()
	if e != nil {
		exitOnError(e)
	}
	// XXX until sync is done
	exitOnError(fmt.Errorf("bug - expected error on sync here"))
	// XXX until sync is done
	if !ok {
		exitOnError(fmt.Errorf("bug - ok should be true if Sync returned nil error"))
	}

	println("------------")
	// error on close here as we have pending ops
	if e := idx.Close(); e == nil {
		exitOnError(fmt.Errorf("bug - expected error on close here"))
	}
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
