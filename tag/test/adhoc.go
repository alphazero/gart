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

	var fname = "tagmap.dat"

	var create bool
	if fi, _ := os.Stat(fname); fi == nil {
		create = true
	}
	tagmap, e := tag.LoadMap(fname, create)
	if e != nil {
		fmt.Printf("%s\n", e)
		os.Exit(1)
	}
	fmt.Printf("%s\n", tagmap)

	var tagNames = []string{"review", "日本語", "unread", "pdf", "distributed-systems", "Pdf", "استاد محمد لطفی"}

	fmt.Println("/// add tags ///////////////////")
	for _, tag := range tagNames {
		if ok, e := tagmap.Add(tag); ok {
			fmt.Printf("op: add %q\n\t%s\n", tag, tagmap)
		} else if e == nil {
			fmt.Printf("add %q - it already exists\n", tag)
		} else {
			fmt.Printf("err - %s\n", e)
		}
		fmt.Println("--")
	}

	fmt.Println("/// update refcnts /////////////")
	for _, tag := range tagNames {
		if refcnt, e := tagmap.IncrRefcnt(tag); e != nil {
			fmt.Printf("err - %s\n", e)
		} else {
			fmt.Printf("op: incr-refcnt retval:%d %q\n\t%s\n", refcnt, tag, tagmap)
		}
		fmt.Println("--")
	}
}
