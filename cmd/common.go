// Doost

package main

import (
	"os"
	"path/filepath"
)

func cmdName() string {
	if len(os.Args) < 1 {
		panic("bug -- os.Args is zerolen")
	}
	return filepath.Base(os.Args[0])
}
