// Doost!

package test

import (
	"math/rand"

	"github.com/alphazero/gart/syslib/bitmap"
)

// NewRandomBitmap creates a new bitmap.Wahl bitmap with a random mix of FILL{0,1}
// and TILE blocks. The new bitmap is compressed.
// TODO optimize this by using wahlWriter.
func NewRandomWahl(rnd *rand.Rand, max uint) *bitmap.Wahl {
	var w = bitmap.NewWahl()
	var bits []uint
	for i := uint(0); i < max; {
		var m uint
		switch typ := rnd.Intn(12); {
		case typ < 8:
			m = uint(31 * uint(rnd.Intn(7)))
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
