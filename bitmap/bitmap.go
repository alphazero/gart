// Doost!

package bitmap

import (
	"fmt"
	"sort"
)

/// Bitmap ////////////////////////////////////////////////////////////////////

type Bitmap []byte // REVU rename to bitmap

type Ops interface { // REVU: maybe better to rename to Bitmap see ^^
	AnySet(bits ...int) bool
	AllSet(bits ...int) bool
	NoneSet(bits ...int) bool // REVU is this just !AllSet() but reads better
}

//    0        1        2        3        4      ... byte
// 7      0 7      0 7      0 7      0 7      0  ... actual byte bit order
// +------+ +------+ +------+ +------+ +------+
// |      | |      | |      | |      | |      |
// +------+ +------+ +------+ +------+ +------+
// x---*--- x*-----* x---*--- x----*-- x-*-----     group hi bit x is never set
//     4     9    15     20        29    34         bits
//
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

// byte aligned variant of WAH
func (v Bitmap) Compress() CompressedBitmap {
	return CompressedBitmap(compress(v))
}

func (v Bitmap) AllSet(bits ...int) bool {
	return allSet(v, bits...)
}

func (v Bitmap) NoneSet(bits ...int) bool {
	return !allSet(v, bits...)
}

func (v Bitmap) AnySet(bits ...int) bool {
	return anySet(v, bits...)
}

func (v Bitmap) String() (s string) {
	return SprintBuf([]byte(v))
}

func (v Bitmap) Debug() {
	DisplayBuf("bitmap", []byte(v))
}

/// CompressedBitmap //////////////////////////////////////////////////////////

// Byte aligned variant of WAH bitmap compression
type CompressedBitmap []byte

func (v CompressedBitmap) Decompress() Bitmap {
	return Bitmap(decompress(v))
}

func (v CompressedBitmap) AnySet(bits ...int) bool {
	return anySet(v, bits...)
}

func (v CompressedBitmap) AllSet(bits ...int) bool {
	return allSet(v, bits...)
}

func (v CompressedBitmap) NoneSet(bits ...int) bool {
	return !allSet(v, bits...)
}

func (v CompressedBitmap) String() (s string) {
	return SprintBuf([]byte(v))
}

func (v CompressedBitmap) Debug() {
	DisplayBuf("compressed", []byte(v))
}

/// santa's little helpers /////////////////////////////////////////////////////

func SprintBuf(buf []byte) (s string) {
	for _, b := range buf {
		s += fmt.Sprintf(" %08b", b)
	}
	return
}

// TODO rename DebugBuf
func DisplayBuf(label string, buf []byte) {
	fmt.Printf("%s\n%s\n", label, SprintBuf(buf))
}
