// Doost !

package main

import (
	"fmt"
	"io"
	"math/bits"
	"os"
	"sort"
	"unsafe"
)

// 11111110110111001011101010011000 01110110010101000011001000010000
//
// 0                                1                                ... byte
// 31                             0 31                             0 ... byte's bit
// +-------x-------x-------x------+ +-------x-------x-------x------+
// |                              | |                              |
// +-------x-------x-------x------+ +-------x-------x-------x------+
// 31      24      16      8      0 63      56      48      40    32 ... bitset bit

var (
	ErrInvalidBitnum  = fmt.Errorf("invalid bit number")
	ErrInvalidArg     = fmt.Errorf("invalid arg")
	ErrOverflow       = fmt.Errorf("data overflows the provided buffer")
	ErrNotImplemented = fmt.Errorf("function not implemented")
)

type Wahl struct {
	arr []uint32
}

func (w *Wahl) Encode(buf []byte) error {
	var wlen = len(w.arr)
	if len(buf) < (wlen << 2) {
		return ErrInvalidArg
	}
	for i := 0; i < wlen; i++ {
		*(*uint32)(unsafe.Pointer(&buf[i<<2])) = w.arr[i]
	}
	return nil
}

func (w *Wahl) Decode(buf []byte) error {
	if len(buf) < 4 {
		return ErrInvalidArg // minimum length
	}
	w.arr = make([]uint32, len(buf)>>2)
	for i := 0; i < len(w.arr); i++ {
		w.arr[i] = *(*uint32)(unsafe.Pointer(&buf[i<<2]))
	}
	return nil
}

func (w *Wahl) Len() int { return len(w.arr) }

func (wah Wahl) Print(w io.Writer) {
	for _, n := range wah.arr {
		fmt.Fprintf(w, "%032b ", bits.Reverse32(n))
	}
	fmt.Fprintf(w, "\n")
}

// REVU should call Set bits.
// TODO New from array (buf or blocks) or just New
func NewWahl(bits ...int) (*Wahl, error) {
	// sort and verify min
	sort.Ints(bits)
	if bits[0] == 0 {
		return nil, ErrInvalidBitnum
	}
	// allocate to hold up to max bitnum in bits
	var wah Wahl
	maxbn := bits[len(bits)-1]
	wlen := (maxbn / 31) + 1
	wah.arr = make([]uint32, wlen)

	for _, bitnum := range bits {
		var bite = bitnum / 31
		var bit = uint(bitnum % 31)
		wah.arr[bite] |= 1 << (bit & 0x1f)
		// shift := (bit & 0x1f)
		// fmt.Printf("bitnum: %d bite:%d bit:%d shift:%d\n", bitnum, bite, bit, shift)
	}
	return &wah, nil
}

func (w *Wahl) Set(bits ...int) {
	sort.Ints(bits)
	// REVU New() should be reconsidered and refactored into Set

	// general approach:
	// as long as bits[i] < w.Max() we are either
	// - setting a bit in an existing tile
	// - splitting a fill-0 block into fill-0-pre, new-tile, fill-0-rem
	// - nop in a fill-1 block.
	/*
		bitmax := bits[len(bits)-1]
		wmax := w.Max()
		if bitmax > w.Max() {
			// REVU requires more thought
		}
	*/
}

// Returns the maximum bit position in bitmap.
func (w *Wahl) Max() int {
	var max int
	for _, b := range w.arr {
		var n int // runlen
		switch {
		case b>>31 == 0x1: // fill
			n = int(b & 0x3fffffff)
		case b>>31 == 0: // tile
			n = 1
		}
		max += (31 * n)
	}
	return max
}

// DecompressTo decompresses the Wahl bitmap by writing directly to the given
// array. The array size must be sufficient to hold the decompressed bitmpa, or
// ErrOverflow is returned.
//
// Function returns the number of uint32 blocks written, and errors, if any.
func (w *Wahl) DecompressTo(buf []uint32) (int, error) {
	return 0, ErrNotImplemented
}

// Decompress decompresses the Wahl bitmap, modifying the receiver. Required
// block storage for the decompressed bitmap is internally allocated.
//
// Function returns a bool indicating if it was modified.
func (w *Wahl) Decompress() bool {
	var wlen = len(w.arr)

	// trivial cases
	if wlen <= 1 {
		return false
	}

	var makefill = func(v uint32, n int) []uint32 {
		a := make([]uint32, n)
		for i := 0; i < n; i++ {
			a[i] = v
		}
		return a
	}
	// Bitmap is decompressed into block array.
	// i indexes the compressed, and j the decompressed blocks.
	const fill_0, fill_1 uint32 = 0x0, 0x7FFFFFFF
	var blocks []uint32
	var i, j int
	for i < wlen {
		var fillblocks []uint32
		var x = w.arr[i]
		switch {
		case x>>30 == 0x3: // fill 1
			n := int(x & 0x3fffffff)
			fmt.Printf("fill-1 block[%d]: %032b - runlen:%d\n", i, x, n)
			fillblocks = makefill(fill_1, n)
			i++
			j += n
		case x>>30 == 0x2: // fill 0
			n := int(x & 0x3fffffff)
			fmt.Printf("fill-1 block[%d]: %032b - runlen:%d\n", i, x, n)
			fillblocks = makefill(fill_0, n)
			i++
			j += n
		case x>>31 == 0: // tile
			fillblocks = []uint32{x}
			fmt.Printf("tile   block[%d]: %032b\n", i, x)
			i++
			j++
		}
		blocks = append(blocks, fillblocks...)
	}
	w.arr = blocks

	return true
}

func (w *Wahl) Compress() bool {
	//	var nopmap [1]byte
	var wlen = len(w.arr)

	// trivial cases
	if wlen <= 1 {
		return false
	}

	// we have a bitmap of at least 2 blocks, so we can meaningfully
	// compress it. we compress in-place.
	// i: index of block to consider for compression
	// j: index of the last (possibly) rewritten block
	//                    j
	// [ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ] ...
	//                             i
	// j may be equal to i if no (further) compression can be done.

	// little helper returns the number of blocks with value v
	var runlen = func(i int, v uint32) int {
		n := 1
		for i+n < wlen && n < 0x3fffffff {
			if w.arr[i+n] != v {
				return n
			}
			n++
		}
		return n
	}
	var i, j int
	for i < wlen {
		var x = w.arr[i]
		switch {
		case x == 0: // all 0 tile
			n := runlen(i, x)
			fmt.Printf("at i:%d j:%d 0 tile - runlen:%d ", i, j, n)
			fill := 0x80000000 | uint32(n)
			fmt.Printf("%032b\n", fill)
			w.arr[j] = 0x80000000 | uint32(n)
			i += n
			j++
		case x == 0x7fffffff: // all 1 tile
			n := runlen(i, x)
			fmt.Printf("at i:%d j:%d 1 tile - runlen:%d ", i, j, n)
			fill := 0xc0000000 | uint32(n)
			fmt.Printf("%032b\n", fill)
			w.arr[j] = 0xc0000000 | uint32(n)
			i += n
			j++
		case x>>31 == 1: // already fill block
			fallthrough
		default: // specific non-monotonic bit pattern tile block
			w.arr[j] = w.arr[i]
			i++
			j++
		}
	}
	// trim (maybe)
	if j < wlen {
		w.arr = w.arr[:j]
		return true
	}
	return false
}

/// adhoc test /////////////////////////////////////////////////////////////////

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// to try:
	// - find optimal way to use []int32 for wah
	// - sketch out Wahl 32-bit encoding, compression, and logical ops

	bits := []int{1, 2, 30, 32, 60, 65, 1024}
	onerun := make([]int, 333)
	for i := 0; i < len(onerun); i++ {
		onerun[i] = 1132 + i
	}
	bits = append(bits, onerun...)
	for i := 0; i < len(onerun); i++ {
		onerun[i] = 2018 + i
	}
	bits = append(bits, onerun...)
	onerun = make([]int, 777)
	for i := 0; i < len(onerun); i++ {
		onerun[i] = 33329 + i
	}
	bits = append(bits, onerun...)

	wah, e := NewWahl(bits...)
	if e != nil {
		exitOnError(e)
	}
	//	arr.Set(0, 8, 16, 24, 31, 32, 40, 48, 56, 63, 64)
	wah.Print(os.Stdout)

	// test unsafe codec
	fmt.Println("-- test encode -- ")
	var buf = make([]byte, 3+wah.Len()*4)

	if e := wah.Encode(buf); e != nil {
		exitOnError(e)
	}
	for _, b := range buf {
		fmt.Printf("%08b ", b)
	}
	fmt.Println()

	fmt.Println("-- test decode -- ")
	var wah2 Wahl
	e = wah2.Decode(buf)
	if e != nil {
		exitOnError(e)
	}
	wah2.Print(os.Stdout)

	fmt.Println("-- test compress -- ")
	var wlen = wah.Len()
	if wah.Compress() {
		fmt.Printf("compressed from %d to %d blocks\n", wlen, wah.Len())
		wah.Print(os.Stdout)
	}

	fmt.Println("-- test decompress -- ")
	wlen = wah.Len()
	if wah.Decompress() {
		fmt.Printf("decompressed from %d to %d blocks\n", wlen, wah.Len())
		wah.Print(os.Stdout)
	}

	fmt.Println("-- test max -- ")
	wah, e = NewWahl(1, 1024, 99999)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("max: %d len:%d\n", wah.Max(), wah.Len())
	wah.Compress()
	fmt.Printf("max: %d len:%d\n", wah.Max(), wah.Len())

}

func exitOnError(e error) {
	fmt.Printf("err - %v\n", e)
	os.Exit(1)
}

// fedcba98                         76543210
