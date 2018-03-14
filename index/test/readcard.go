// Doost!

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alphazero/gart/index"
)

const file = "/Users/alphazero/Code/go/src/gart/process.go"
const file1 = "/Users/alphazero/Code/go/src/github.com/alphazero/gart/process.go"
const file2 = "/usr/local/go/src/alphazero/gart/process.go"
const garthome = "/Users/alphazero/.gart"

func main() {

	if len(os.Args) == 1 {
		exitOnUsage()
	}
	var file = os.Args[1]

	fmt.Printf("Salaam!\n")

	// only use absolute paths
	if !filepath.IsAbs(file) {
		wd, e := os.Getwd()
		if e != nil {
			panic(e)
		}
		file = filepath.Join(wd, file)
	}

	oid, e := index.ObjectId(file)
	if e != nil {
		exitOnError(e)
	}

	card, e := index.GetCard(garthome, oid)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("%s\n", card.DebugStr())

	// card.Save should return false
	// keep this only to insure above is true
	doSave(card)
}

func doSave(card index.Card) {
	// write it
	ok, e := card.Save()
	if e != nil {
		exitOnError(e)
	}
	if ok {
		fmt.Printf("%s\n", card.DebugStr())
		panic(fmt.Sprintf("test: wrote: %t\n", ok))
	}
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "err - %s\n", e)
	os.Exit(1)
}

func exitOnUsage() {
	fmt.Fprintf(os.Stderr, "usage: readcard <for-filename>\n")
	os.Exit(2)
}
