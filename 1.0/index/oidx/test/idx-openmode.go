// Doost!

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alphazero/gart/index/oidx"
)

var modeStr string

func init() {
	flag.StringVar(&modeStr, "op", modeStr, " r:Read w:Write v:verify")
}

const garthome = "/Users/alphazero/.gart"

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")
	flag.Parse()

	modeStr = strings.ToLower(modeStr)

	var opMode oidx.OpMode
	switch modeStr {
	case "r":
		opMode = oidx.Read
	case "w":
		opMode = oidx.Write
	default:
		exitOnError(fmt.Errorf("unknown opMode: %q\n", modeStr))
	}
	idx, e := oidx.OpenIndex(garthome, opMode)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("Openned oidx index in mode: %s\n%v\n", opMode, idx)

	updated, e := idx.CloseIndex()
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("Closed oidx index -- updated: %t\n", updated)
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
