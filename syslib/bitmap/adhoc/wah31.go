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

var _ = bits.Reverse32

func (w *Wahl) Len() int { return len(w.arr) }

func (wah Wahl) Print(w io.Writer) {
	var binfo = func(b uint32, max0 int) (int, int, int) {
		n := 1
		if b>>31 > 0 {
			n = int(b & 0x3fffffff)
		}
		return max0 + 1, max0 + (n * 31), n
	}

	var min, max, n int = -1, -1, 0 //= binfo(0, 0)
	for _, b := range wah.arr {
		min, max, n = binfo(b, max)
		fmt.Fprintf(w, "%06d       %06d       %06d ", min, n, max) // bits.Reverse32(n))
	}
	fmt.Fprintf(w, "\n")
	for range wah.arr {
		fmt.Fprintf(w, "..........1.........2.........3. ") // bits.Reverse32(n))
	}
	fmt.Fprintf(w, "\n")
	for range wah.arr {
		fmt.Fprintf(w, ".123456789.123456789.123456789.1 ") // bits.Reverse32(n))
	}
	fmt.Fprintf(w, "\n")
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

// Set is used to
// 	(a) create new bitmaps when gart is adding object with a new tag.
//      most likely case is a sequence of 'bits' (corresonding to object keys)
//      from n -> n+k. But this is not necessarily always true. (a 'new' file
//		may in fact be an existing object with a new tag.)
//
//	(b) a single bit is being set. This is a very likely case, even in case of
//		aggregate adds unless top-level gart batches the updates to the index.
//
//	(c) general case: a random selection of bits, not necessarily in sort order,
//		are being set.
//
// Function may allocate new blocks before a final decompression at the end.
func (w *Wahl) Set(bits ...int) {
	if len(bits) > 1 {
		sort.Ints(bits)
	}

	// if the maximum bitnum in bits is > w.Max(), then add the necessary blocks.
	var wmax = w.Max() // (initial) maximum bit position in bitmap
	var bitsmax = bits[len(bits)-1]
	if bitsmax > wmax {
		nblks := make([]uint32, ((bitsmax-wmax)/31)+1)
		w.arr = append(w.arr, nblks...)
	}
	// Note we may still have to add more blocks if any fill-0 blocks need to
	// be split but bitsmax is guaranteed to be in range of the bitmap and the
	// updated wmax will -not- be affected.
	wmax = w.Max() // update it. it will be >= bitsmax

	// helper function returns range of the block given the prior
	// block's range-max and runlen
	var brange = func(b uint32, max0 int) (int, int, int) {
		n := 1
		if b>>31 > 0 {
			n = int(b & 0x3fffffff)
		}
		return max0 + 1, max0 + (n * 31), n
	}

	// since we've sorted the bits arg, an index i of wahl blocks will only
	// move forward.
	var i int // current block
	// [min, max] bitrange of current block and its runlength
	var bmin, bmax, rn = brange(w.arr[i], 0)
	for _, bitnum := range bits {
		for bitnum > bmax {
			i++
			bmin, bmax, rn = brange(w.arr[i], bmax)
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
			rn_a := (bitnum - bmin) / 31 // REVU if rn_a == 0
			rn_z := rn - rn_a - 1        // REVU if rn_z == 0
			switch {
			case rn_a == 0: // split in 2 - set bit in 1st block
				var bit = uint(bitnum % 31)
				w.arr[i] = 1 << (bit & 0x1f)
				arr := make([]uint32, len(w.arr)+1)
				copy(arr, w.arr[:i+1])
				arr[i+1] = 0x80000000 | uint32(rn_z) // assert (rn_z == rn - 1)
				copy(arr[i+2:], w.arr[i+1:])
				w.arr = arr
				bmax = bmin + 30
			case rn_z == 0: // split in 2 - set bit in 2nd block
				arr := make([]uint32, len(w.arr)+1)
				w.arr[i] = 0x80000000 | uint32(rn_a) // asert (rn_a == rn - 1)
				copy(arr, w.arr[:i+1])
				var bit = uint(bitnum % 31)
				arr[i+1] = 1 << (bit & 0x1f)
				copy(arr[i+2:], w.arr[i+1:])
				w.arr = arr
				// update block info - current is the new tile added
				i++
				bmin += (rn_a * 31)
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
				bmin += (rn_a * 31)
				bmax = bmin + 30
			}
		case block>>30 == 0x3: // fill-1 is already set, next bit!
			continue
		default: // tile needs to have bitpos 'bitnum' set
			var bit = uint(bitnum % 31)
			w.arr[i] |= 1 << (bit & 0x1f)
		}
	}
}

// Returns the maximum bit position in bitmap. This is simply the
// number of decompressed blocks x 31. Function does -not- decompress and is
// side-effect free.
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
	/*
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

	*/
	fmt.Println("-- test max -- ")
	//	wahl, e := NewWahl(31, 32, 1024)
	wahl, e := NewWahl(1024)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("max: %d len:%d\n", wahl.Max(), wahl.Len())
	wahl.Compress()
	fmt.Printf("max: %d len:%d\n", wahl.Max(), wahl.Len())
	wahl.Print(os.Stdout)
	fmt.Println("-- test set -- ")
	wahl.Set(5, 333, 1000, 1027)
	fmt.Printf("max: %d len:%d\n", wahl.Max(), wahl.Len())
	wahl.Print(os.Stdout)
}

func exitOnError(e error) {
	fmt.Printf("err - %v\n", e)
	os.Exit(1)
}
