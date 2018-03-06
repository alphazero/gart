// Doost!

package bitmap

import (
	"fmt"
	"sort"
)

/// Bitmap ////////////////////////////////////////////////////////////////////

type Bitmap []byte

//    0        1        2        3        4      ... encode 7 group
// 7      0 7      0 7      0 7      0 7      0  ... actual byte bit order
// +------+ +------+ +------+ +------+ +------+
// |      | |      | |      | |      | |      |
// +------+ +------+ +------+ +------+ +------+
// x---*--- x*-----* x---*--- x----*-- x-*-----     group hi bit x is never set
//     4     9    15     20        29    34         bits
func Build(bits ...int) Bitmap {
	sort.IntSlice(bits).Sort()
	max := bits[len(bits)-1]
	var bitmap = Bitmap(make([]byte, (max>>3)+1))

	for _, bit := range bits {
		i := uint(bit)
		b := i >> 3       // b: byte number [0,..)
		n := i - (b << 3) // n: nth bit in the byte
		if n == 0 {
			panic(fmt.Errorf("bug - bitmap.Build: mod-8 congruent bit index"))
		}
		bitmap[b] |= 0x80 >> n
	}
	return bitmap
}

func (v Bitmap) String() (s string) {
	return SprintBuf([]byte(v))
}

// byte aligned variant of WAH
func (v Bitmap) Compress() CompressedBitmap {
	panic("Bitmap.Compress: not implemented")
}

/// CompressedBitmap //////////////////////////////////////////////////////////

// Byte aligned variant of WAH bitmap compression
type CompressedBitmap []byte

func (v CompressedBitmap) MatchAny(bit ...int) bool {
	panic("CompressedBitmap.MatchAny: not implemented")
}

func (v CompressedBitmap) MatchAll(bit ...int) bool {
	panic("CompressedBitmap.MatchAll: not implemented")
}

func (v CompressedBitmap) String() (s string) {
	return SprintBuf([]byte(v))
}

/// santa's little helpers /////////////////////////////////////////////////////

func SprintBuf(buf []byte) (s string) {
	for _, b := range buf {
		s += fmt.Sprintf(" %08b", b)
	}
	return
}
