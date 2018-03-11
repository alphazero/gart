// Doost!

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/index/card"
)

const fname = "/Users/alphazero/Code/go/src/gart/process.go"
const fname1 = "/Users/alphazero/Code/go/src/github.com/alphazero/gart/process.go"
const fname2 = "/usr/local/go/src/alphazero/gart/process.go"
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
	crd, e := card.New(oid, fname, tags, systemics)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("%s\n", crd.DebugStr())

	doSave(crd)

	// sleep to check updated timestamp change
	time.Sleep(time.Second * 2)
	add := []string{"", fname1, fname, fname2}

	for _, p := range add {
		doAdd(crd, p)
	}

	doSave(crd)

	doRemove(crd, fname2)
	doRemove(crd, "")
	doRemove(crd, fname1)

	doSave(crd)

	/// check Read
	crd1 := doRead(crd.Oid())
	doRemove(crd1, fname)

	fmt.Printf("& Salaam!\n")
}
func doRead(oid index.OID) index.Card {

	cfile := cardfile(&oid)
	crd, e := card.Read(cfile)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("read: %q\n", cfile)
	fmt.Printf("%s\n", crd.DebugStr())
	return crd
}

func doSave(card index.Card) {
	// write it
	var oid = card.Oid()
	cfile := cardfile(&oid)
	if e := card.Save(cfile); e != nil {
		exitOnError(e)
	}
	fmt.Printf("wrote: %q\n", cfile)
	fmt.Printf("%s\n", card.DebugStr())
}

func doRemove(card index.Card, s string) {
	fmt.Printf("/// remove path /// %q\n", s)
	ok, e := card.RemovePath(s)
	if e != nil {
		fmt.Printf("err - %s\n", e)
		return
	}
	if ok {
		fmt.Printf("removed %q\n", s)
		//		fmt.Printf("%s\n", card.DebugStr())
	} else {
		fmt.Printf("%q not found\n", s)
	}
}

func doAdd(card index.Card, s string) {
	fmt.Printf("/// add path /// %q\n", s)
	ok, e := card.AddPath(s)
	if e != nil {
		fmt.Printf("err - %s\n", e)
		return
	}
	if ok {
		fmt.Printf("added %q\n", s)
		//		fmt.Printf("%s\n", card.DebugStr())
	} else {
		fmt.Printf("existing %q not added\n", s)
	}
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
