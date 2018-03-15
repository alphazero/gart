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
	fmt.Printf("bitmap: %s\n\n", bmap)

	bah := bmap.Compress()
	fmt.Printf("compressed: %s\n\n", bah)

	bmap2 := bah.Decompress()
	fmt.Printf("decompressed: %s\n\n", bmap2)

	var qbits []int

	// match using the uncompressed bitmap - this should return TRUE
	qbits = []int{1, 7, 9, 15, 95, 803, 805, 1025}
	result := bmap.AllSet(qbits...)
	fmt.Printf("result:%t\n\tqbits:%d\n\t bits:%d\n\n", result, qbits, bits)

	println("-------")

	// match using the bah07 compressed bitmap - this should return TRUE
	qbits = []int{1, 7, 9, 15, 95, 803, 805, 1025}
	result = bah.AllSet(qbits...)
	fmt.Printf("result:%t\n\tqbits:%d\n\t bits:%d\n\n", result, qbits, bits)

	println("-------")

	// match using the bah07 compressed bitmap -- this should return FALSE
	qbits = []int{1, 7, 9, 15, 85, 803, 805, 1025}
	result = bah.AllSet(qbits...)
	fmt.Printf("result:%t\n\tqbits:%d\n\t bits:%d\n\n", result, qbits, bits)

	println("-------")

	// expect false
	qbits = []int{3, 6, 10, 14, 87, 801, 807, 102}
	result = bah.NoneSet(qbits...)
	fmt.Printf("non-set result:%t\n\tqbits:%d\n\n", result, qbits)

	println("-------")

	// expect true (803)
	qbits = []int{3, 6, 10, 14, 87, 803, 807, 102}
	result = bah.AnySet(qbits...)
	fmt.Printf("any-set result:%t\n\tqbits:%d\n\n", result, qbits)

	return nil
}
