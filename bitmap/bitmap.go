// Doost!

package bitmap

import (
	"fmt"
	"sort"
)

/// Bitmap //////////////////////////////////////////////////////////////////////

// Generalized bitmap ops (compressed or not).
// For all functions in this interface, the in-arg bits array
// must be in ascending sort order.
type Bitmap interface {
	// IFF uncompressed returns a compressed version, otherwise returns itself
	Compress() Bitmap
	// IFF compressed returns a decompressed version, otherwise returns itself
	Decompress() Bitmap
	// Returns true if Bitmap is compressed
	Compressed() bool
	// Convenience method
	Bytes() []byte
	// Returns true if any of the bits are set
	AnySet(bits ...int) bool
	// Returns true if all of the bits set
	AllSet(bits ...int) bool
	// Returns true if none of the bits are set
	NoneSet(bits ...int) bool
	// Returns true if bitmap changed
	Set(bits ...int) bool
	// String rep
	String() string
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

/// bitmap_t ////////////////////////////////////////////////////////////////////

type bitmap_t []byte

func New(buf []byte) bitmap_t {
	var bm = make([]byte, len(buf))
	copy(bm, buf)
	return bitmap_t(bm)
}

func (v bitmap_t) Bytes() []byte { return v }

func (v bitmap_t) Decompress() Bitmap { return v }

func (v bitmap_t) Compressed() bool { return false }

// byte aligned variant of WAH
func (v bitmap_t) Compress() Bitmap {
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

func (v bitmap_t) Set(bits ...int) bool {
	return set(v, bits...)
}

func (v bitmap_t) String() string {
	return SprintBuf(v)
}

func (v bitmap_t) Debug() {
	DisplayBuf("bitmap", v)
}

/// compressed_t //////////////////////////////////////////////////////////

// Byte aligned variant of WAH bitmap compression
type compressed_t []byte

func NewCompressed(buf []byte) Bitmap {
	var bm = make([]byte, len(buf))
	copy(bm, buf)
	return compressed_t(bm)
}

func (v compressed_t) Bytes() []byte { return v }

func (v compressed_t) Compressed() bool { return false }

func (v compressed_t) Compress() Bitmap { return v }

func (v compressed_t) Decompress() Bitmap {
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

func (v compressed_t) Set(bits ...int) bool {
	return set(v, bits...)
}

func (v compressed_t) String() (s string) {
	return SprintBuf(v)
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
