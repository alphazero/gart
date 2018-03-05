// Doost!

package main

import (
	"fmt"
	"github.com/alphazero/gart/tag"
	"os"
	"time"
)

func main() {
	fmt.Printf("Salaam! %d\n", time.Now().UnixNano())

	var fname = "tags.dat"

	var create bool
	if fi, _ := os.Stat(fname); fi == nil {
		create = true
	}
	tags, e := tag.LoadMap(fname, create)
	if e != nil {
		fmt.Printf("%s\n", e)
		os.Exit(1)
	}
	fmt.Printf("%s\n", tags)

	var tagNames = []string{"review", "unread", "pdf", "distributed-systems", "Pdf"}
	for _, tag := range tagNames {
		if tags.Add(tag) {
			fmt.Printf("op: add %q\n\t%s\n", tag, tags)
		} else {
			fmt.Printf("add %q - it already exists\n", tag)
		}
	}
}
