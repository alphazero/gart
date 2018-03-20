// Doost!

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/index/oidx"
)

//var filename = "/Users/alphazero/.gart/index/objects.idx"

var gartHome = "/Users/alphazero/.gart"
var op string
var debug bool

func init() {
	flag.StringVar(&op, "op", op, "op: 'r', 'w', 'q', 'qrange")
	flag.BoolVar(&debug, "debug", debug, "debug")
}

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	flag.Parse()
	op = strings.ToLower(op)

	var e error
	switch op {
	case "r":
		e = readFromIt()
	case "w":
		e = writeToIt(333)
	case "q":
		keys := []uint64{57, 17, 22, 18, 34}
		e = queryIt(keys...)
	case "qrange":
		keys := []uint64{57, 17, 22, 18, 34, 777}
		e = queryIt(keys...)
	default:
		exitOnError(fmt.Errorf("invalid op: %q", op))
	}
	if e != nil {
		exitOnError(e)
	}
}

func readFromIt() error {
	mmf, e := oidx.OpenIndex(gartHome, oidx.Read)
	if e != nil {
		return e
	}
	// gart process complete:
	defer mmf.CloseIndex()

	mmf.DevDebug()
	return nil
}

func writeToIt(items int) error {
	// gart process prepare:
	mmf, e := oidx.OpenIndex(gartHome, oidx.Write)
	if e != nil {
		return e
	}
	defer mmf.CloseIndex()

	// gart-add:
	for i := 0; i < items; i++ {
		oid := oidForObject(i)
		//		oid := digest.Sum([]byte(fmt.Sprintf("object-%d", i)))
		if e := mmf.AddObject(oid[:]); e != nil {
			fmt.Printf("err - writeToIt: %v", e)
			return e
		}
	}
	return nil
}

func oidForObject(n int) [index.OidSize]byte {
	return digest.Sum([]byte(fmt.Sprintf("object-%d", n)))
}

func queryIt(keys ...uint64) error {
	// gart process prepare:
	mmf, e := oidx.OpenIndex(gartHome, oidx.Read)
	if e != nil {
		return e
	}
	defer mmf.CloseIndex()
	// gart process prepare:

	// Note! Lookup sorts the keys as side-effect!
	oids, e := mmf.Lookup(keys...)
	if e != nil {
		return e
	}

	// display them
	if debug {
		fmt.Printf("Lookup results:\n")
		for i, oid := range oids {
			fmt.Printf("[%03d]: oid: %x\n", i, oid)
		}
		fmt.Println("\t---")
		fmt.Printf("Expected set (not in sort order):\n")
		for _, key := range keys {
			fmt.Printf("key %03d =>  %x\n", key, oidForObject(int(key)))
		}
	}
	return nil
}

func exitOnError(e error) {
	fmt.Printf("err - %s\n", e)
	os.Exit(1)
}
