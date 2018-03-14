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

	card, newCard, updated, e := index.AddOrUpdateCard(garthome, oid, file, tags, systemics)
	if e != nil {
		exitOnError(e)
	}
	if newCard {
		fmt.Printf("test: new card - revision: %d\n", card.Revision())
	}
	fmt.Printf("test: updated: %t\n", updated)
	fmt.Printf("%s\n", card.DebugStr())

	// NOTE for newbasic.go card is always saved on 'index.AddOrUpdate' so
	// card.Save always returns false
	doSave(card)
}

func doSave(card index.Card) {
	// write it
	ok, e := card.Save()
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("test: wrote: %t\n", ok)
	if ok {
		fmt.Printf("%s\n", card.DebugStr())
	}
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
