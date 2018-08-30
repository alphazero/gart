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

// TODO a typical form of bitmap will have clusters of 1s in form of
//      Tile[000..111..] Fill-1 [rlen-cluster-core] Tile[1111..00..] Fill-0[rlen-gap]
//      This will happen, for example, when adding a set of new files w/ the same tags.
func NewRandomClustered(rnd *rand.Rand, max uint) *Wahl {
	// basically pick Tile-init, Tile-end, and cluster-len and gap-len randomly.
	panic("not implemented")
}

// TODO a pathological case of bitmap is where we have long runs of tiles which are
//      almost all 1s but not Fills. This can happen as either result of query ops
//      (for an intermediate result map) or for tags that cross cut across semantic
//      tags. For example, we may add a set of files with many extensions, but the
//      files are added in a way that specific extensions are distributed across the
//      set.
//      Fill-x[]Tile[..]Tile[..].....Tile[...]Fill-x[]
//      Effectively, these are bitmaps that resist compression and processing them
//      incurs both the cost of WAHL (no O(1) addressing of bits) and poor performance
//      in bitwise ops (since bitwise op cost is directly related to number of words
//      that are processed).

func NewRandomPathological(rnd *rand.Rand, max uint) *Wahl {
	// basically pick tile-cluster-len and intersperse with random Fill of rand x..
	panic("not implemented")
}
