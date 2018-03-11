// Doost!

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/index/card"
)

const fname = "/Users/alphazero/Code/go/src/gart/process.go"
const fname1 = "/Users/alphazero/Code/go/src/github.com/alphazero/gart/process.go"
const path = "/Users/alphazero/Code/go/src/gart/index/card/test"

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

	var tags = []byte{0x7f, 0x81, 0x02}
	var systemics = []byte{0x7f}

	//	var cfile = cardfile(oid)
	card, e := card.New(oid, fname, tags, systemics)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("%s\n", card.DebugStr())

	// write it
	cfile := cardfile(oid)
	if e := card.Save(cfile); e != nil {
		exitOnError(e)
	}
	fmt.Printf("wrote: %q\n", cfile)

	fmt.Println("/// add a path //////////////////")
	ok, e := card.AddPath(fname1)
	if e != nil {
		exitOnError(e)
	}
	if ok {
		fmt.Printf("added %q\n", fname1)
		fmt.Printf("%s\n", card.DebugStr())
	} else {
		fmt.Printf("existing %q not added\n", fname1)
	}

	fmt.Printf("& Salaam!\n")
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
