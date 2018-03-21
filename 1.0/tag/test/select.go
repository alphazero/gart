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

	tagmap, e := tag.LoadMap(fname, false)
	if e != nil {
		fmt.Printf("%s\n", e)
		os.Exit(1)
	}
	fmt.Printf("%s\n", tagmap)

	fmt.Println("/// select tags ////////////////")

	var query = []string{"Love", "Doost", "nf 1", "日本語", "mp3", "nf 2", "programming-language", "PDF", "استاد محمد لطفی"}

	ids, excluded := tagmap.SelectTags(query)
	fmt.Printf("ids:      %d \n", ids)
	fmt.Printf("excluded: %q \n", excluded)
}
