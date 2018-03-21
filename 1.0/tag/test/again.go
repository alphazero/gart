// Doost!

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alphazero/gart/tag"
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

	if ok, e := tagmap.Sync(); ok {
		panic("bug - sync() of unmodified tagmap returned true")
	} else if e != nil {
		panic("bug - sync() of unmodified tagmap returned an error")
	}

	var tagNames = []string{"love", "Doost", "日本", "mp4", "mp3", "programming-language", "PLT"}

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

	if ok, e := tagmap.Sync(); e != nil {
		panic(e)
	} else if !ok {
		panic("bug - sync() of modified tagmap returned false")
	}

	fmt.Println("sync ok!")
}
