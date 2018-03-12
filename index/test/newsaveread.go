// Doost!

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/index"
)

const fname = "/Users/alphazero/Code/go/src/gart/process.go"
const path = "/Users/alphazero/Code/go/src/gart/index/test"

func cardfile(oid *index.OID) string {
	s := fmt.Sprintf("%x.card", *oid)
	fmt.Printf("debug - s: %s\n", s)
	return filepath.Join(path, s)
}

func main() {

	fmt.Printf("Salaam!\n")

	md, e := digest.SumFile(fname)
	if e != nil {
		exitOnError(e)
	}
	oid, e := index.NewOid(md)
	if e != nil {
		exitOnError(e)
	}

	var tags = bitmap.NewCompressed([]byte{0x7f, 0x81, 0x02})
	var systemics = bitmap.NewCompressed([]byte{0x7f})

	crd, e := index.NewCard(oid, fname, tags, systemics)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("%s\n", crd.DebugStr())

	doSaveCard(crd)

	/// check Read
	doRead(crd.Oid())

	fmt.Printf("& Salaam!\n")
}
func doRead(oid index.OID) index.Card {

	cfile := cardfile(&oid)
	crd, e := index.ReadCard(cfile)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("read: %q\n", cfile)
	fmt.Printf("%s\n", crd.DebugStr())
	return crd
}

func doSaveCard(card index.Card) {
	// write it
	var oid = card.Oid()
	cfile := cardfile(&oid)
	if e := card.Save(cfile); e != nil {
		exitOnError(e)
	}
	fmt.Printf("wrote: %q\n", cfile)
	fmt.Printf("%s\n", card.DebugStr())
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
