// Doost!

package bitmap

import (
	"fmt"
	"sort"
)

/// bitmap_t ////////////////////////////////////////////////////////////////////

type bitmap_t []byte

// Generalized bitmap ops (compressed or not).
// For all functions in this interface, the in-arg bits array
// must be in ascending sort order.
type Bitmap interface {
	AnySet(bits ...int) bool
	AllSet(bits ...int) bool
	NoneSet(bits ...int) bool
}

//    0        1        2        3        4      ... byte
// 7      0 7      0 7      0 7      0 7      0  ... actual byte bit order
// +------+ +------+ +------+ +------+ +------+
// |      | |      | |      | |      | |      |
// +------+ +------+ +------+ +------+ +------+
// x---*--- x*-----* x---*--- x----*-- x-*-----     group hi bit x is never set
//     4     9    15     20        29    34         bits
//
func Build(bits ...int) bitmap_t {
	sort.IntSlice(bits).Sort()
	max := bits[len(bits)-1]
	var bitmap = bitmap_t(make([]byte, (max>>3)+1))

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
func (v bitmap_t) Compress() compressed_t {
	return compressed_t(compress(v))
}

// Returns true if all bits are sest in the bitmap
func (v bitmap_t) AllSet(bits ...int) bool {
	return allSet(v, bits...)
}

// Returns true if none of the bits are sest in the bitmap
func (v bitmap_t) NoneSet(bits ...int) bool {
	return noneSet(v, bits...)
}

// Returns true if any of the bits are sest in the bitmap
func (v bitmap_t) AnySet(bits ...int) bool {
	return anySet(v, bits...)
}

func (v bitmap_t) String() (s string) {
	return SprintBuf([]byte(v))
}

func (v bitmap_t) Debug() {
	DisplayBuf("bitmap", []byte(v))
}

/// compressed_t //////////////////////////////////////////////////////////

// Byte aligned variant of WAH bitmap compression
type compressed_t []byte

func (v compressed_t) Decompress() bitmap_t {
	return bitmap_t(decompress(v))
}

// Returns true if any of the bits are sest in the bitmap
func (v compressed_t) AnySet(bits ...int) bool {
	return anySet(v, bits...)
}

// Returns true if all bits are sest in the bitmap
func (v compressed_t) AllSet(bits ...int) bool {
	return allSet(v, bits...)
}

// Returns true if of the bits bits are sest in the bitmap
func (v compressed_t) NoneSet(bits ...int) bool {
	return noneSet(v, bits...)
}

func (v compressed_t) String() (s string) {
	return SprintBuf([]byte(v))
}

func (v compressed_t) Debug() {
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
