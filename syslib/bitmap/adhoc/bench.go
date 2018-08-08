// Doost!

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"

	"github.com/alphazero/gart/syslib/bench"
	"github.com/alphazero/gart/syslib/bitmap"
	//	"github.com/alphazero/gart/syslib/bitmap/test"
)

var (
	seed int64
	max  uint = 1 << 16
	reps int  = 1000
)

var rnd *rand.Rand

func init() {
	flag.Int64Var(&seed, "seed", seed, "seed")
	flag.UintVar(&max, "max", max, "max bit num")
	flag.IntVar(&reps, "reps", reps, "reps")
}

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")
	flag.Parse()
	fmt.Printf("bench wahl: max: %d - seed: %d\n", max, seed)

	rnd = rand.New(rand.NewSource(seed))

	var tstamp = bench.NewTimestamp()
	var w_0, w_1 *bitmap.Wahl
	w_0 = bitmap.NewRandomWahl(rnd, max)
	w_1 = bitmap.NewRandomWahl(rnd, max)
	//	w_0 = test.NewRandomWahl(rnd, max)
	//	w_1 = test.NewRandomWahl(rnd, max)
	//	w_0.Print(os.Stdout)
	//	w_1.Print(os.Stdout)
	tstamp.Mark("2 newRandomBitmap")

	for i := 0; i < reps; i++ {
		bitmap.And(w_0, w_1)
	}
	tstamp.MarkN("AND", reps)
	for i := 0; i < reps; i++ {
		bitmap.Or(w_0, w_1)
	}
	tstamp.MarkN("OR ", reps)
	for i := 0; i < reps; i++ {
		bitmap.Xor(w_0, w_1)
	}
	tstamp.MarkN("XOR", reps)

}

func exitOnError(e error) {
	fmt.Printf("err - %v\n", e)
	os.Exit(1)
}
