// Doost !

package main

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/syslib/bitmap"
)

/// adhoc test /////////////////////////////////////////////////////////////////

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// to try:
	// - find optimal way to use []int32 for wah
	// - sketch out Wahl 32-bit encoding, compression, and logical ops
	// Thank you FRIEND! Done!
	tryUncompressed()
	tryCompressed()
}

func exitOnError(e error) {
	fmt.Printf("err - %v\n", e)
	os.Exit(1)
}

func tryCompressed() {
	// lotsofones are bits in range (1000, 1332)
	lotsofones := make([]uint, 333)
	for i := 0; i < len(lotsofones); i++ {
		lotsofones[i] = uint(i + 1000)
	}

	var wahl_1 = bitmap.NewWahl()
	wahl_1.Set(0, 30, 63, 93)
	wahl_1.Set(lotsofones[:333]...)
	wahl_1.Bits().Print(os.Stdout)
	wahl_1.Compress()
	wahl_1.Print(os.Stdout)

	var wahl_2 = bitmap.NewWahl()
	wahl_2.Set(0, 1, 29, 31, 93, 124, 155, 185, 186, 1000, 1001, 1003, 1007, 2309, 2311)
	wahl_2.Set(lotsofones[:111]...)
	wahl_2.Bits().Print(os.Stdout)
	wahl_2.Compress()
	wahl_2.Print(os.Stdout)

	// test AND
	fmt.Printf("=== TEST AND ====================\n")
	wahl_1_and_2, e := wahl_1.And(wahl_2)
	if e != nil {
		exitOnError(e)
	}
	wahl_1_and_2.Bits().Print(os.Stdout)
	wahl_1_and_2.Print(os.Stdout)
	verifyAnd(wahl_1, wahl_2, wahl_1_and_2)

	// test OR
	fmt.Printf("=== TEST OR =====================\n")
	wahl_1_or_2, e := wahl_1.Or(wahl_2)
	if e != nil {
		exitOnError(e)
	}
	wahl_1_or_2.Bits().Print(os.Stdout)
	wahl_1_or_2.Print(os.Stdout)
	verifyOr(wahl_1, wahl_2, wahl_1_or_2)

	// test Clear
	fmt.Printf("=== TEST Clear ==================\n")
	fmt.Printf("=== clear anded bitmap ==========\n")
	wahl_1_and_2.Clear(0)                // clear tile
	wahl_1_and_2.Clear(95)               // nop tile
	wahl_1_and_2.Clear(222)              // clear fill-0 (nop)
	wahl_1_and_2.Clear(1023, 1025, 1027) // clear fill-1
	wahl_1_and_2.Compress()
	wahl_1_and_2.Bits().Print(os.Stdout)
	wahl_1_and_2.Print(os.Stdout)

	fmt.Printf("=== clear or'd bitmap -==========\n")
	wahl_1_or_2.Clear(1, 30)                        // clear tile
	wahl_1_or_2.Clear(1023, 1052, 1111, 1300, 1302) // clear fill-1
	wahl_1_or_2.Compress()
	wahl_1_or_2.Bits().Print(os.Stdout)
	wahl_1_or_2.Print(os.Stdout)

	return
}

func smallerFirst(a, b []int) ([]int, []int) {
	if len(a) < len(b) {
		return a, b
	}
	return b, a
}
func mapArray(a []int) map[int]bool {
	a_map := make(map[int]bool)
	for _, v := range a {
		a_map[v] = true
	}
	return a_map
}
func verifyAnd(a, b, and *bitmap.Wahl) {
	a_map := mapArray(a.Bits())
	b_map := mapArray(b.Bits())
	for _, bit := range and.Bits() {
		// bit must be in both maps for AND
		if !(a_map[bit] && b_map[bit]) {
			fmt.Printf("AND: bit %d is not in both maps\n", bit)
		}
	}
}

func verifyOr(a, b, or *bitmap.Wahl) {
	a_map := mapArray(a.Bits())
	b_map := mapArray(b.Bits())
	for _, bit := range or.Bits() {
		// bit must be in both maps for AND
		if !(a_map[bit] || b_map[bit]) {
			fmt.Printf("OR: bit %d is not in either maps\n", bit)
		}
	}
}

func tryUncompressed() {
	// lotsofones are bits in range (1000, 1332)
	lotsofones := make([]uint, 333)
	for i := 0; i < len(lotsofones); i++ {
		lotsofones[i] = uint(i + 1000)
	}

	var wahl_1 = bitmap.NewWahl()
	wahl_1.Set(0, 30, 63, 93)
	wahl_1.Set(lotsofones[:33]...)
	wahl_1.Bits().Print(os.Stdout)
	wahl_1.Print(os.Stdout)

	var wahl_2 = bitmap.NewWahl()
	wahl_2.Set(0, 1, 29, 31, 93, 124, 155, 185, 186, 1000, 1001, 1003, 1007, 2309, 2311)
	wahl_2.Bits().Print(os.Stdout)
	wahl_2.Print(os.Stdout)

	wahl_1_and_2, e := wahl_1.And(wahl_2)
	if e != nil {
		exitOnError(e)
	}
	wahl_1_and_2.Bits().Print(os.Stdout)
	wahl_1_and_2.Print(os.Stdout)

	return
}
