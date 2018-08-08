// Doost!

package test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/alphazero/gart/syslib/bitmap"
)

var bench_rnd *rand.Rand

func init() {
	bench_rnd = rand.New(rand.NewSource(0)) // deterministic
}

func BenchmarkBitwiseOps(b *testing.B) {
	// maximum number of wahl blocks, e.g.
	// sizes[i] * 31 => max set bit position.
	var sizes []uint
	for pow := uint(10); pow < 26; pow += 2 {
		sizes = append(sizes, 1<<pow)
	}

	for _, maxBit := range sizes {
		// generate random bitmaps for given maxbit
		maxBit := maxBit
		sname := fmt.Sprintf("[%d bits]", maxBit)
		w_0 := bitmap.NewRandomWahl(bench_rnd, maxBit)
		w_1 := bitmap.NewRandomWahl(bench_rnd, maxBit)

		//		w_0.Print(os.Stdout)
		//		w_1.Print(os.Stdout)
		fmt.Fprintf(os.Stdout, "for %d lens (%d %d)\n", maxBit, w_0.Len(), w_1.Len())
		// bench all ops for maxbit sized bitmaps.
		for _, op := range bitwiseOps {
			op := op
			bname := fmt.Sprintf("%s %s", sname, op.name)
			b.Run(bname, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					op.fn(w_0, w_1)
				}
			})
		}
		println()
	}
}
