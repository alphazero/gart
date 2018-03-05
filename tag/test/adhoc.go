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

	var tagNames = []string{"review", "日本語", "unread", "pdf", "distributed-systems", "Pdf"}
	for _, tag := range tagNames {
		if ok, e := tags.Add(tag); ok {
			fmt.Printf("op: add %q\n\t%s\n", tag, tags)
			// TODO tags.Select(tag)
		} else if e == nil {
			fmt.Printf("add %q - it already exists\n", tag)
		} else {
			fmt.Printf("err - %s\n", e)
		}
	}
}
