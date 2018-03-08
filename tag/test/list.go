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

	var fname = "/Users/alphazero/.gart/tags/TAGS"

	tagmap, e := tag.LoadMap(fname, false)
	if e != nil {
		fmt.Printf("%s\n", e)
		os.Exit(1)
	}
	fmt.Printf("%s\n", tagmap)

	fmt.Println("/// list tags //////////////////")
	tags := tagmap.Tags()
	for _, tag := range tags {
		fmt.Printf("%s\n", tag.Debug())
	}
}
