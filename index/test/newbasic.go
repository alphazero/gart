// Doost!

package main

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/index"
)

const file = "/Users/alphazero/Code/go/src/gart/process.go"
const file1 = "/Users/alphazero/Code/go/src/github.com/alphazero/gart/process.go"
const file2 = "/usr/local/go/src/alphazero/gart/process.go"
const garthome = "/Users/alphazero/.gart"

func main() {

	fmt.Printf("Salaam!\n")

	oid, e := index.ObjectId(file)
	if e != nil {
		exitOnError(e)
	}

	var tags = bitmap.NewCompressed([]byte{0x7f, 0x81, 0x02})
	var systemics = bitmap.NewCompressed([]byte{0x7f})

	//	var cfile = cardfile(oid)
	card, updated, e := index.AddOrUpdateCard(garthome, oid, file, tags, systemics)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("updated: %t\n", updated)
	fmt.Printf("%s\n", card.DebugStr())

	doSave(card)
}

func doSave(card index.Card) {
	// write it
	ok, e := card.Save()
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("wrote: %t\n", ok)
	fmt.Printf("%s\n", card.DebugStr())
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
