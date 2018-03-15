// Doost!

package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/alphazero/gart/bitmap"
)

var rnd = rand.New(rand.NewSource(333))

func main() {
	fmt.Println("Salaam! %s", time.Now())

	if e := run(); e != nil {
		fmt.Printf("err - %s\n", e)
	}
}

func run() error {
	var bits = []int{1, 7, 9, 15, 81, 87, 95, 803, 804, 805, 1023, 1025}
	fmt.Printf("bits: %d\n\n", bits)

	bmap := bitmap.Build(bits...)
	bmap_copy := bitmap.Build(bits...)
	bah := bmap.Compress()
	bah_copy := bmap.Compress()

	assertNil(bmap.AssertSize(1, 255))
	assertNil(bmap_copy.AssertSize(1, 255))
	assertNil(bah.AssertSize(1, 255))
	assertNil(bah_copy.AssertSize(1, 255))

	fmt.Printf("equal: %t \n\tmap:%s \n\tmap_copy:%s\n\n", bmap.IsEqual(bmap_copy), bmap, bmap_copy)
	fmt.Printf("equal: %t \n\tmap:%s \n\tbah:%s\n\n", bmap.IsEqual(bah), bmap, bah)

	fmt.Printf("equal: %t \n\tbah:%s \n\tmap:%s\n\n", bah.IsEqual(bmap), bah, bmap)
	fmt.Printf("equal: %t \n\tbah:%s \n\tbah:%s\n\n", bah.IsEqual(bah_copy), bah, bah_copy)

	println("-------")

	return nil
}

func assertNil(v interface{}) {
	if v != nil {
		panic("assert failed")
	}
}
