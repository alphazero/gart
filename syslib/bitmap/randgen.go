// Doost!

package bitmap

import (
	"math/rand"
)

// NewRandomBitmap creates a new bitmap.Wahl bitmap with a random mix of FILL{0,1}
// and TILE blocks. The new bitmap is compressed.
// TODO optimize this by using wahlWriter.
// TODO appendWriter needs to be package public or randgen part of bitmap package.
// REVU part of package is simpler.
func NewRandomWahl(rnd *rand.Rand, max uint) *Wahl {
	w := newWriter(nil)

	for i := uint(0); i < max; {
		var m uint // m is cap on bit range (i, i+m]
		switch typ := rnd.Intn(100); {
		case typ < 50:
			x := rnd.Intn(int(max)>>10) + 1
			m = uint(31 * uint(x))
			if rnd.Int()&1 == 0 {
				w.writeN(0x7fffffff, x)
			} else {
				w.writeN(0, x)
			}
			i += m
		default:
			x := rnd.Intn(12) // larger x longer run of tiles
			m = uint(31 * uint(x))
			for k := 0; k < x; k++ {
				w.writeN(uint32(rnd.Intn(0x7fffffff)), 1)
				i += 31
			}
		}
	}
	return w.done()
}
