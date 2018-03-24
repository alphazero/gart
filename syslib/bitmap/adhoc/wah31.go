// Doost !

package main

import (
	"fmt"
	"io"
	"math/bits"
	"os"
	"unsafe"

	"github.com/alphazero/gart/syslib/sort"
)

var (
	ErrInvalidBitnum  = fmt.Errorf("invalid bit number")
	ErrInvalidArg     = fmt.Errorf("invalid arg")
	ErrOverflow       = fmt.Errorf("data overflows the provided buffer")
	ErrNotImplemented = fmt.Errorf("function not implemented")
)

/// wahl block (helper) ////////////////////////////////////////////////////////

type wahlBlock struct {
	val  uint32
	fill bool
	fval int
	rlen int // 1 for tiles (to be consistent)
}

func (b wahlBlock) String() string {
	var revbit = bits.Reverse32(b.val)
	var typ = "tile"
	if b.fill {
		if b.fval == 0 {
			typ = "fill-0"
		} else {
			typ = "fill-1"
		}
		return fmt.Sprintf("%030b %02b %-6s (%d)", revbit>>2, revbit&0x3, typ, b.rlen)
	}
	return fmt.Sprintf("%031b-  %-6s", revbit>>1, typ)
}

func WahlBlock(v uint32) wahlBlock {
	var block = wahlBlock{v, false, 0, 1} // assume tile
	switch {
	case v>>31 == 0: // tile
		return block
	case v>>30 == 0x3: // fill 1
		block.fval = 1
	case v>>30 == 0x1: // fill 0
		block.fval = 0
	}
	block.fill = true
	block.rlen = int(v & 0x3fffffff)
	return block
}

/// WπAπAπL /////////////////////////////////////////////////////////////////

// 0                                1                                ... byte
// 31                             0 31                             0 ... byte's bit
// +x------x-------x-------x------+ +x------x-------x-------x------+
// |                              | |                              |
// +x------x-------x-------x------+ +-------x-------x-------x------+
//  30     24      16      8      0  63     56      48      40    32 ... bitset bit
//

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

// REVU also New from array (buf or blocks) or just New

// Allocates a new, zerovalue, Wahl object.
func NewWahl() *Wahl { return &Wahl{[]uint32{0}} }

// Allocates a new (compressed) Wahl bitmap with the given initial bits.
func NewWahlInit(bits ...uint) *Wahl {
	w := NewWahl()

	maxbn := bits[len(bits)-1]
	wlen := (maxbn / 31) + 1
	w.arr = make([]uint32, wlen)

	w.Set(bits...)
	return w
}

// Set sets the given 'bits' of the bitmap. It is irrelevant whether the bitmap
// is in compressed or decompressed state.
//
// Set method will perform a final compress before returning.
func (w *Wahl) Set(bits ...uint) {
	if len(bits) > 1 {
		sort.Uints(bits)
	}

	// add additional blocks to w.arr if necessary
	var wmax = w.Max() // (initial) maximum bit position in bitmap
	var bitsmax = int(bits[len(bits)-1])
	if bitsmax > wmax {
		nblks := make([]uint32, ((bitsmax-wmax)/31)+1)
		w.arr = append(w.arr, nblks...)
	}
	// We may still have to add more blocks if any fill-0 blocks need to
	// be split but bitsmax is guaranteed to be in range of the bitmap and the
	// updated wmax will -not- be affected.
	wmax = w.Max() // update it. it will be >= bitsmax

	// returns range of the block 'b' & the prior max
	var brange = func(b uint32, max0 int) (uint, uint, int) {
		min := uint(max0) + 1
		n := 1
		if b>>31 > 0 {
			n = int(b & 0x3fffffff)
		}
		return min, min + (uint(n) * 31) - 1, n
	}

	var i int // current block
	var bmin, bmax, rn = brange(w.arr[i], -1)
	for _, bitnum := range bits {
		for bitnum > bmax {
			i++
			bmin, bmax, rn = brange(w.arr[i], int(bmax))
		}
		switch block := w.arr[i]; {
		case block>>30 == 0x2:
			// fill-0 needs to be split (into 3 or 2 blocks) or changed into a tile
			if rn == 1 { // change to tile
				var bit = uint(bitnum % 31)
				w.arr[i] = 1 << (bit & 0x1f)
				continue
			}
			// splits
			rn_a := int(bitnum-bmin) / 31
			rn_z := rn - rn_a - 1
			switch {
			case rn_a == 0: // split in 2 - set bit in 1st block
				var bit = uint(bitnum % 31)
				w.arr[i] = 1 << (bit & 0x1f)
				arr := make([]uint32, len(w.arr)+1)
				copy(arr, w.arr[:i+1])
				arr[i+1] = 0x80000000 | uint32(rn_z)
				copy(arr[i+2:], w.arr[i+1:])
				w.arr = arr
				// update block info
				bmax = bmin + 30
			case rn_z == 0: // split in 2 - set bit in 2nd block
				arr := make([]uint32, len(w.arr)+1)
				w.arr[i] = 0x80000000 | uint32(rn_a)
				copy(arr, w.arr[:i+1])
				var bit = uint(bitnum % 31)
				arr[i+1] = 1 << (bit & 0x1f)
				copy(arr[i+2:], w.arr[i+1:])
				w.arr = arr
				// update block info - current is the new tile added
				i++
				bmin += uint(rn_a * 31)
				bmax = bmin + 30
			default: // split in 3 - set bit in middle block
				arr := make([]uint32, len(w.arr)+2)
				w.arr[i] = 0x80000000 | uint32(rn_a)
				copy(arr, w.arr[:i+1])
				var bit = uint(bitnum % 31)
				arr[i+1] = 1 << (bit & 0x1f)
				arr[i+2] = 0x80000000 | uint32(rn_z)
				copy(arr[i+3:], w.arr[i+1:])
				w.arr = arr
				// update block info - current is the new tile added
				i++
				bmin += uint(rn_a * 31)
				bmax = bmin + 30
			}
		case block>>30 == 0x3: // fill-1 is already set, next bit!
			continue
		default: // tile needs to have bitpos 'bitnum' set
			var bit = uint(bitnum % 31)
			w.arr[i] |= 1 << (bit & 0x1f)
		}
	}
	w.Compress()
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
			fillblocks = makefill(fill_1, n)
			i++
			j += n
		case x>>30 == 0x2: // fill 0
			n := int(x & 0x3fffffff)
			fillblocks = makefill(fill_0, n)
			i++
			j += n
		case x>>31 == 0: // tile
			fillblocks = []uint32{x}
			i++
			j++
		}
		blocks = append(blocks, fillblocks...)
	}
	w.arr = blocks

	return true
}

// Compress will (further) compress the bitmap.
// Returns true if bitmap size is reduced.
func (w *Wahl) Compress() bool {
	var wlen = len(w.arr)

	// trivial cases
	if wlen <= 1 {
		return false
	}

	// little helper returns the number consequitive of blocks with value v
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

	// Compress in-place.
	// i: index of block to consider for compression
	// j: index of the last (possibly) rewritten block
	var i, j int
	for i < wlen {
		var x = w.arr[i]
		switch {
		case x == 0: // all 0 tile
			n := runlen(i, x)
			w.arr[j] = 0x80000000 | uint32(n)
			i += n
			j++
		case x == 0x7fffffff: // all 1 tile
			n := runlen(i, x)
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

// Returns the number of blocks
func (w *Wahl) Len() int { return len(w.arr) }

// Returns the maximal bit position
func (w *Wahl) Max() int {
	var max int = -1
	if e := w.apply(maxBitsVisitor(&max)); e != nil {
		panic(fmt.Errorf("bug - Wahl.Max: %v", e))
	}
	return max
}

// Note that bits are reversed and printed LSB -> MSB
func (w Wahl) Print(writer io.Writer) {
	if e := w.apply(printVisitor(writer)); e != nil {
		panic(fmt.Errorf("bug - Wahl.Print: %v", e))
	}
	fmt.Fprintf(writer, "\n")
}

// Note that bits are reversed and printed LSB -> MSB
func (w Wahl) Debug(writer io.Writer) {
	if e := w.apply(debugVisitor(writer)); e != nil {
		panic(fmt.Errorf("bug - Wahl.Print: %v", e))
	}
	fmt.Fprintf(writer, "\n")
}

/// Wahl visitors //////////////////////////////////////////////////////////////

// Visit function for Wahl.
type visitFn func(bn int, val uint32) (done bool, err error)

func maxBitsVisitor(max *int) visitFn {
	return func(bnum int, bval uint32) (bool, error) {
		*max += int(31 * WahlBlock(bval).rlen)
		return false, nil
	}
}

func printVisitor(w io.Writer) visitFn {
	return func(bnum int, bval uint32) (bool, error) {
		block := WahlBlock(bval)
		fmt.Fprintf(w, "[%4d]: %s\n", bnum, block)
		return false, nil
	}
}

func debugVisitor(w io.Writer) visitFn {
	return func(bnum int, bval uint32) (bool, error) {
		block := WahlBlock(bval)
		if block.fill {
		} else {
			fmt.Fprintf(w, "       01234567890123456789012345678901\n")
			fmt.Fprintf(w, "       0---------1---------2---------3-\n")
		}
		fmt.Fprintf(w, "[%4d]:%s\n", bnum, block)
		fmt.Fprintf(w, "\n")
		return false, nil
	}
}

// apply will walk the blocks and apply the visit function in sequence.
// Iteration is stopped on completion or error by the visit func (which is
// returned).
func (w *Wahl) apply(visit visitFn) error {
	for i, block := range w.arr {
		done, e := visit(i, block)
		if e != nil {
			return e
		}
		if done {
			return nil
		}
	}
	return nil
}

/// adhoc test /////////////////////////////////////////////////////////////////

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// to try:
	// - find optimal way to use []int32 for wah
	// - sketch out Wahl 32-bit encoding, compression, and logical ops

	fmt.Println("-- test NewWah -- ")
	var wahl0 = NewWahl()
	wahl0.Print(os.Stdout)

	fmt.Println("-- test Set -- ")
	lotsofones := make([]uint, 333)
	for i := 0; i < len(lotsofones); i++ {
		lotsofones[i] = uint(i + 999)
	}
	wahl0.Set(lotsofones...)
	wahl0.Set(0, 61, 222, 1024)
	wahl0.Decompress()
	wahl0.Print(os.Stdout)
	wahl0.Compress()
	wahl0.Print(os.Stdout)
	return

	fmt.Println("-- test NewWahInit-- ")
	//	var wahl = NewWahlInit(1, 7, 11, 13, 17, 19, 23, 29, 30, 31, 37, 2309, 2311)
	// TODO fix BUG when bitnum is multiple of 31
	var wahl = NewWahlInit(0, 124, 155, 185, 186, 2309, 2311) // 7, 7, 11, 13, 17, 19, 23, 29, 30, 37, 2309, 2311)
	//wahl.Decompress()                                         // decompress as Wahl is compressed by default
	wahl.Print(os.Stdout)
	return
	// end BUG

	fmt.Println("-- test compress -- ")
	fmt.Printf("inital     - max:         %d len:%d\n", wahl.Max(), wahl.Len())
	wahl.Compress()
	fmt.Printf("compressed - max:         %d len:%d\n", wahl.Max(), wahl.Len())
	wahl.Print(os.Stdout)

	fmt.Println("-- test Set -- ")
	wahl.Set(5, 333, 1000, 1027, 1132)
	fmt.Printf("max: %d len:%d\n", wahl.Max(), wahl.Len())
	wahl.Debug(os.Stdout)
}

func exitOnError(e error) {
	fmt.Printf("err - %v\n", e)
	os.Exit(1)
}
