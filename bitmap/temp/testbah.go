// Doost!

package main

import (
	"fmt"
	"github.com/alphazero/gart/bitmap"
	"math/rand"
	"time"
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
	fmt.Printf("bitmap: %s\n\n", bmap)

	bah := bmap.Compress()
	fmt.Printf("compressed: %s\n\n", bah)

	bmap2 := bah.Decompress()
	fmt.Printf("decompressed: %s\n\n", bmap2)

	var qbits = []int{1, 7, 9, 15, 95, 803, 805, 1025}
	allSet := bah.MatchAll(qbits...)
	fmt.Printf("allSet:%t\n\tqbits:%d\n\t bits:%d\n\n", allSet, qbits, bits)

	println("-------")

	qbits = []int{1, 7, 9, 15, 85, 803, 805, 1025}
	allSet = bah.MatchAll(qbits...)
	fmt.Printf("allSet:%t\n\tqbits:%d\n\t bits:%d\n\n", allSet, qbits, bits)

	return nil
}
