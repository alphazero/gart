// Doost!

package main

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/index/oidx"
)

const garthome = "/Users/alphazero/.gart"

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	idx, e := oidx.OpenIndex(garthome, oidx.Read)
	if e != nil {
		exitOnError(e)
	}

	var keyset = []uint64{57, 22, 17, 18, 34}
	oidsetBytes, e := idx.Lookup(keyset...)
	if e != nil {
		exitOnError(e)
	}
	var oids = make([]*index.OID, len(oidsetBytes))
	for i, dat := range oidsetBytes {
		oids[i] = index.NewOid(dat)
	}

	if e := idx.Close(); e != nil {
		exitOnError(e)
	}
	fmt.Printf("Closed object.idx\n")
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
