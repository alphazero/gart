// Doost !

package main

import (
	"fmt"
	"os"

	. "github.com/alphazero/gart/syslib/bitmap"
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

	var wahl_1 = NewWahl()
	wahl_1.Set(0, 30, 63, 93)
	wahl_1.Set(lotsofones[:33]...)
	wahl_1.Bits().Print(os.Stdout)
	wahl_1.Compress()
	wahl_1.Print(os.Stdout)

	var wahl_2 = NewWahl()
	wahl_2.Set(0, 1, 29, 31, 93, 124, 155, 185, 186, 1000, 1001, 1003, 1007, 2309, 2311)
	wahl_2.Bits().Print(os.Stdout)
	wahl_2.Compress()
	wahl_2.Print(os.Stdout)

	wahl_1_and_2, e := wahl_1.And(wahl_2)
	if e != nil {
		exitOnError(e)
	}
	wahl_1_and_2.Bits().Print(os.Stdout)
	wahl_1_and_2.Print(os.Stdout)

	return
}
func tryUncompressed() {
	// lotsofones are bits in range (1000, 1332)
	lotsofones := make([]uint, 333)
	for i := 0; i < len(lotsofones); i++ {
		lotsofones[i] = uint(i + 1000)
	}

	var wahl_1 = NewWahl()
	wahl_1.Set(0, 30, 63, 93)
	wahl_1.Set(lotsofones[:33]...)
	wahl_1.Bits().Print(os.Stdout)
	wahl_1.Print(os.Stdout)

	var wahl_2 = NewWahl()
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
