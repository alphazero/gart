// Doost!

package main

import (
	"fmt"
	"math/rand"
	"os"
	//	"time"
	"flag"

	"github.com/alphazero/gart/syslib/bench"
	"github.com/alphazero/gart/syslib/bitmap"
	//	"github.com/alphazero/gart/syslib/errors"
)

var (
	seed int64
	max  uint = 1 << 16
)

var rnd *rand.Rand

func init() {
	flag.Int64Var(&seed, "seed", seed, "seed")
	flag.UintVar(&max, "max", max, "max bit num")
}

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")
	flag.Parse()
	fmt.Printf("bench wahl: max: %d - seed: %d\n", max, seed)

	rnd = rand.New(rand.NewSource(seed))

	var tstamp = bench.NewTimestamp()
	var w_0, w_1 *bitmap.Wahl
	w_0 = newRandomBitmap(max)
	w_1 = newRandomBitmap(max)
	tstamp.Mark("2 newRandomBitmap")

	const reps = 100
	for i := 0; i < reps; i++ {
		bitmap.And(w_0, w_1)
	}
	tstamp.MarkN("AND", reps)
	for i := 0; i < reps; i++ {
		bitmap.Or(w_0, w_1)
	}
	tstamp.MarkN("OR", reps)
	for i := 0; i < reps; i++ {
		bitmap.Xor(w_0, w_1)
	}
	tstamp.MarkN("XOR", reps)

}

func newRandomBitmap(max uint) *bitmap.Wahl {
	var w = bitmap.NewWahl()
	var bits []uint
	for i := uint(0); i < max; {
		var m uint
		switch typ := rnd.Intn(12); {
		case typ < 8:
			m = uint(31 << uint(rnd.Intn(7)))
			if rnd.Int()&1 == 0 {
				for j := uint(0); j < m; j++ {
					bits = append(bits, i+j)
				}
			}
		default:
			m = uint(31 << uint(rnd.Intn(3)))
			for j := uint(0); j < m; j++ {
				if rnd.Intn(100) > 50 {
					bits = append(bits, i+j)
				}
			}
		}
		i += m
	}
	w.Set(bits...)
	w.Compress()
	return w
}

func exitOnError(e error) {
	fmt.Printf("err - %v\n", e)
	os.Exit(1)
}
