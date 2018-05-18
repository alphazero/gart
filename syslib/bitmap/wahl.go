// Doost!
//
//	                      ~ π ~ W A H L ~ π ~
//
// ~!! Ya Wadud !!~ ~!! Ya Wahab !!~ ~!! Ya Wahid !!~ ~!! Ya Wajib !!~
// ~!! Ya Wakil !!~ ~!! Ya Waali !!~ ~!! Ya Wali !!~  ~!! Ya Warith !!~
//                          ~!! Ya Wasi !!~
//
// ~!! Ya ALLAH !!~ ~!! Ya Akhar !!~ ~!! Ya Adl !!~  ~!! Ya Afu! !!~
// ~!! Ya Ali !!~  ~!! Ya Alim !!~  ~!! Ya Awwal !!~ ~!! Ya Azim !!~
//                          ~!! Ya Aziz !!~
//
// ~!! Ya Hadi !!~  ~!! Ya Hasib !!~ ~!! Ya Hafiz !!~ ~!! Ya Hakam !!~
// ~!! Ya Hakim !!~ ~!! Ya Halim !!~ ~!! Ya Hamid !!~ ~!! Ya Haqq !!~
//                          ~!! Ya Hay !!~
//
//                          ~!! Ya Latif !!~
//
//                         !!! Only -YOU- !!!
//
// O Loving Kind !  O Bestowing !  O ONE !            O Resourceful !
// O Trustee !      O Protector !  O Loving Friend !  O Final Inheritor !
//             ~!! O Vastness Without Approachable Limits !!~
//
// O GOD !          O Last !           O Just !   O Effacer of Spiritual Error!
// O Most High!     O All Knowing !    O First !  O Tremendous !
//                        ~!! O Mighty Lord !!~
//
// O Guide !    O Perfect Reckoner !     O Preserver !         O Ruler !
// O Wise !     O Indulgent !            O Praise Worthy !     O Truth !
//                        ~!! O Living One !!~
//
//                     ~!! O Sublimely Gentle !!~
//
//                         !!! -Only- YOU !!!
//
// Salaam Samad Sultan of LOVE!

package bitmap

import (
	"fmt"
	"io"
	"math/bits"
	"unsafe"

	//	"github.com/alphazero/gart/syslib/bench"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/sort"
)

/// π WAHL π ////////////////////////////////////////////////////////////////

// WAHL is a Word Aligned Hybride Long (32bit) compressible bitmap. The bits
// are 31-bit encoded into a sequence of 'tiles' (literal bit image) and
// 'fill' blocks (run-length encoding of a consecutive 0s or 1s). The block
// type is distinguished by a control bit at the MSB (bit 31) of the block.
//
// A tile block is a 32-bit word with MSB of 0 indicating the tile type. The
// remaining 31 bits are the literal sequence of the bitmap fragment. The
// positional order corresponds to the bits of the uint32 type. The following
// diagram is an example of a WAHL bitmap that begins with 2 tile blocks.
//
// 0                                  1                                 ... L-word
//   30                            0    30                            0 ... word's bit
// +-x------x-------x-------x------x+ +-x------x-------x-------x------x+
// |0       t i l e   b l o c k     | |0       t i l e   b l o c k     |
// +-x------x-------x-------x------x+ +-x------x-------x-------x------x+
//   30     24      16      8      0    63     56      48      40    32 ... bitset bit
//
// Fill blocks are compressed form encoding of a monotonic sequence of 0s
// or 1s of length equal to a multiple of 31. An MSB of 1 indicates that
// the uint32 word is a fill block. The preceding bit indicates the fill
// sequence value. Bits (0, 30) encode the runlength factor k, with the
// sequence length being equal to k * 31. Thus 1 to 2^30-1 multiples of
// 31 can be represented by a single fill block.
//
// 0                                  1                                 ... L-word
// 3130                            0  3130                            0 ... word's bit
// +xx------------------------------+ +xx-----------------------------x+
// |10  f i l l - 0   b l o c k     | |11  f i l l - 1   b l o c k     |
// +--------------------------------+ +--------------------------------+
//
type Wahl struct {
	arr []uint32
}

// Returns the number of blocks
func (w *Wahl) Len() int { return len(w.arr) }

// Returns the (encoded) size in bytes.
func (w *Wahl) Size() int { return len(w.arr) << 2 }

// Allocates a new, zerovalue, Wahl object.
func NewWahl() *Wahl { return &Wahl{[]uint32{}} }

// Allocates a new (compressed) Wahl bitmap with the given initial bits.
func NewWahlInit(bits ...uint) *Wahl {
	w := NewWahl()

	maxbn := bits[len(bits)-1]
	wlen := (maxbn / 31) + 1
	w.arr = make([]uint32, wlen)

	w.Set(bits...)
	return w
}

// And applies the logical AND operation to the given bitmaps, returning
// the resulting bitmap. The input args are not modified.
//
// Returns nil, error if pair-wise Wahl.And returns any error.
func And(bitmaps ...*Wahl) (*Wahl, error) {
	return bitwise(AndOp, bitmaps...)
}

// Or applies the logical OR operation to the given bitmaps, returning
// the resulting bitmap. The input args are not modified.
//
// Returns nil, error if pair-wise Wahl.Or returns any error.
func Or(bitmaps ...*Wahl) (*Wahl, error) {
	return bitwise(OrOp, bitmaps...)
}

// Xor applies the logical XOR operation to the given bitmaps, returning
// the resulting bitmap. The input args are not modified.
//
// Returns nil, error if pair-wise Wahl.Xor returns any error.
func Xor(bitmaps ...*Wahl) (*Wahl, error) {
	return bitwise(XorOp, bitmaps...)
}

func bitwise(op bitwiseOp, bitmaps ...*Wahl) (*Wahl, error) {
	if len(bitmaps) == 0 {
		return NewWahl(), nil
	}

	var resmap = bitmaps[0]
	var e error
	for _, bmap := range bitmaps[1:] {
		resmap, e = resmap.bitwise(op, bmap)
		if e != nil {
			return nil, e
		}
	}
	return resmap, nil
}

// Set sets the given 'bits' of the bitmap. It is irrelevant whether the bitmap
// Set sets the given 'bits' of the bitmap. It is irrelevant whether the bitmap
// is in compressed or decompressed state.
//
// Set method will not perform a final compress before returning given that
// function's relative computational costs. It is advisable to perform a compress
// after the bitmap setting is done.
func (w *Wahl) Set(bits ...uint) bool {
	return w.set(true, bits...)
}

// Clear sets the given 'bits' of the bitmap to 0. It is irrelevant whether the bitmap
// is in compressed or decompressed state.
//
// Clear method will not perform a final compress before returning given that
// function's relative computational costs. It is advisable to perform a compress
// after the bitmap setting is done.
func (w *Wahl) Clear(bits ...uint) bool {
	return w.set(false, bits...)
}

var clearMask32 = [31]uint32{
	0x7ffffffe,
	0x7ffffffd,
	0x7ffffffb,
	0x7ffffff7,
	0x7fffffef,
	0x7fffffdf,
	0x7fffffbf,
	0x7fffff7f,
	0x7ffffeff,
	0x7ffffdff,
	0x7ffffbff,
	0x7ffff7ff,
	0x7fffefff,
	0x7fffdfff,
	0x7fffbfff,
	0x7fff7fff,
	0x7ffeffff,
	0x7ffdffff,
	0x7ffbffff,
	0x7ff7ffff,
	0x7fefffff,
	0x7fdfffff,
	0x7fbfffff,
	0x7f7fffff,
	0x7effffff,
	0x7dffffff,
	0x7bffffff,
	0x77ffffff,
	0x6fffffff,
	0x5fffffff,
	0x3fffffff,
}

// Set the bits to the given bitval (true=>1, false=>0)
func (w *Wahl) set(bitval bool, bits ...uint) bool {
	var bitslen = len(bits)
	if bitslen == 0 {
		return false
	}
	if bitslen > 1 {
		sort.Uints(bits)
	}

	var setCase, nopCase uint32
	var setter func(bitnum uint) (bval uint32)
	var blockMask uint32
	if bitval { // Set
		setCase = 0x02
		nopCase = 0x03
		blockMask = 0x80000000
		setter = func(bitnum uint) uint32 { return 1 << ((bitnum % 31) % 0x1f) }
	} else { // Clear
		setCase = 0x03
		nopCase = 0x02
		blockMask = 0xc0000000
		setter = func(bitnum uint) uint32 { return clearMask32[(bitnum%31)%0x1f] }
	}

	// add additional blocks to w.arr if necessary
	var wmax = w.Max() // (initial) maximum bit position in bitmap
	var bitsmax = int(bits[bitslen-1])
	if bitsmax > wmax || len(w.arr) == 0 {
		nblks := make([]uint32, ((bitsmax-wmax)/31)+1)
		w.arr = append(w.arr, nblks...)
	}
	// We may still have to add more blocks if any fill-0 blocks need to
	// be split but bitsmax is guaranteed to be in range of the bitmap and the
	// updated wmax will -not- be affected.
	wmax = w.Max() // update it. it will be >= bitsmax

	var i int // current block
	var bmin, bmax, rn = blockRange(w.arr[i], -1)
	for _, bitnum := range bits {
		for bitnum > bmax {
			i++
			bmin, bmax, rn = blockRange(w.arr[i], int(bmax))
		}
		switch block := w.arr[i]; {
		case block>>30 == setCase:
			// fill-x needs to be split (into 3 or 2 blocks) or changed into a tile
			if rn == 1 { // change to tile
				w.arr[i] = setter(bitnum)
				continue
			}
			// splits
			rn_a := int(bitnum-bmin) / 31
			rn_z := rn - rn_a - 1
			switch {
			case rn_a == 0: // split in 2 - set bit in 1st block
				w.arr[i] = setter(bitnum)
				arr := make([]uint32, len(w.arr)+1)
				copy(arr, w.arr[:i+1])
				arr[i+1] = blockMask | uint32(rn_z)
				copy(arr[i+2:], w.arr[i+1:])
				w.arr = arr
				// update block info
				bmax = bmin + 30
			case rn_z == 0: // split in 2 - set bit in 2nd block
				arr := make([]uint32, len(w.arr)+1)
				w.arr[i] = blockMask | uint32(rn_a)
				copy(arr, w.arr[:i+1])
				arr[i+1] = setter(bitnum)
				copy(arr[i+2:], w.arr[i+1:])
				w.arr = arr
				// update block info - current is the new tile added
				i++
				bmin += uint(rn_a * 31)
				bmax = bmin + 30
			default: // split in 3 - set bit in middle block
				arr := make([]uint32, len(w.arr)+2)
				w.arr[i] = blockMask | uint32(rn_a)
				copy(arr, w.arr[:i+1])
				arr[i+1] = setter(bitnum)
				arr[i+2] = blockMask | uint32(rn_z)
				copy(arr[i+3:], w.arr[i+1:])
				w.arr = arr
				// update block info - current is the new tile added
				i++
				bmin += uint(rn_a * 31)
				bmax = bmin + 30
			}
		case block>>30 == nopCase: // fill-x is already bitval filled , next bit!
			continue
		default: // tile needs to have bitpos 'bitnum' set to bitval
			// REVU it is likely faster to just set the bit (even if already set)
			// than branch to check if it needs to be set.
			// Consequence of this is that this function always returns true 'updated'.
			if bitval {
				w.arr[i] |= setter(bitnum)
			} else {
				w.arr[i] &= setter(bitnum)
			}
		}
	}
	return true
}

// DecompressTo decompresses the Wahl bitmap by writing directly to the given
// array. The array size must be sufficient to hold the decompressed bitmpa, or
// ErrOverflow is returned.
//
// Function returns the number of uint32 blocks written, and errors, if any.
func (w *Wahl) DecompressTo(buf []uint32) (int, error) {
	return 0, errors.ErrNotImplemented
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

// Compress will (further) compress the bitmap. Current implementation is a
// 2-pass compressor.
// Returns true if bitmap size is reduced.
func (w *Wahl) Compress() bool {

	/// santa's little helpers //////////////////////////////////////

	// returns number of consecuitive tiles with value v
	var tileMerge = func(i int, v uint32) int {
		n := 1
		for i+n < len(w.arr) && n < 0x3fffffff {
			if w.arr[i+n] != v {
				return n
			}
			n++
		}
		return n
	}
	// returns number of consecuitive fills with value v
	// and the total runlen
	var fillMerge = func(i int, fval int) (int, int) {
		var rn int
		var nb = 0
		for nb+i < len(w.arr) {
			wb := WahlBlock(w.arr[nb+i])
			if !wb.fill || wb.fval != fval {
				return rn, nb
			}
			nb++
			rn += wb.rlen
		}
		return rn, nb
	}

	/// compressor //////////////////////////////////////////////////

	var pass int
	for pass < 2 {
		var wlen = len(w.arr)
		// trivial cases
		if wlen <= 1 {
			return false
		}

		// Compress in-place.
		// i: index of block to consider for compression
		// j: index of the last (possibly) rewritten block
		var i, j int
		for i < wlen {
			var wb = WahlBlock(w.arr[i])
			switch {
			case wb.val == 0: // all 0 tile
				n := tileMerge(i, wb.val)
				w.arr[j] = 0x80000000 | uint32(n)
				i += n
				j++
			case wb.val == 0x7fffffff: // all 1 tile
				n := tileMerge(i, wb.val)
				w.arr[j] = 0xc0000000 | uint32(n)
				i += n
				j++
			case wb.fill:
				rn, bn := fillMerge(i, wb.fval)
				fill := uint32(0x80000000) | uint32(wb.fval<<30) | uint32(rn)
				w.arr[j] = fill
				j++
				i += bn
			default: // specific non-monotonic bit pattern tile block
				w.arr[j] = w.arr[i]
				i++
				j++
			}
		}
		// trim (maybe)
		if j < wlen {
			w.arr = w.arr[:j]
		}
		pass++
	}

	// final remove (maybe) trailing fill-0
	k := len(w.arr) - 1
	wb := WahlBlock(w.arr[k])
	if (wb.fill && wb.fval == 0) || (!wb.fill && wb.val == 0x7fffffff) {
		w.arr = w.arr[:k]
	}

	return false
}

// REVU nest helper in TODO consolidated LogicalOp (And/Or)
// maxInt returns the maximum of inputs (a, b)
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Returns new bitmap that is the logical NOT of the receiver.
// The new has same len and max as the
func (w Wahl) Not() *Wahl {
	var wnot = &Wahl{arr: make([]uint32, len(w.arr))}
	for i, v := range w.arr {
		switch v >> 31 {
		case 1:
			wnot.arr[i] = v ^ 0x40000000
		default:
			wnot.arr[i] = v ^ 0x7FFFFFFF
		}
	}
	return wnot
}

// And appllies the bitwise logical AND, returns result in a newly allocated bitmap.
// Returns ErrInvalidArg if input is nil.
func (w Wahl) And(other *Wahl) (*Wahl, error) {
	return w.bitwise(AndOp, other)
}

// Or applies the bitwise logical OR, returns result in a newly allocated bitmap.
// Returns ErrInvalidArg if input is nil.
func (w Wahl) Or(other *Wahl) (*Wahl, error) {
	return w.bitwise(OrOp, other)
}

// Xor applies the bitwise logical XOR, returns result in a newly allocated bitmap.
// Returns ErrInvalidArg if input is nil.
func (w Wahl) Xor(other *Wahl) (*Wahl, error) {
	return w.bitwise(XorOp, other)
}

type bitwiseOp byte

const (
	_ bitwiseOp = iota
	AndOp
	OrOp
	XorOp
)

func (w1 Wahl) bitwise(op bitwiseOp, w2 *Wahl) (*Wahl, error) {
	if w2 == nil {
		return nil, errors.ErrInvalidArg
	}

	/// blockwise application of ap /////////////////////////////////

	//	t0 := bench.NewTimestamp()
	i, j, k, res, wb1, wb2 := w1.blockwise(op, w2)
	//	t0.Mark("blockwise")

	/// op finalization /////////////////////////////////////////////

	// here we check if any partially processed tail blocks remain
	// we only care if op is XOR | OR and append those directly to blockwise result.
	if op != AndOp {
		//		debug.Printf("tail @ i:%d j:%d k:%d", i, j, k)
		var wlen1, wlen2 int = w1.Len(), w2.Len()
		switch {
		case i >= wlen1 && j >= wlen2:
		case j >= wlen2:
			if wb1.rlen > 0 {
				res[k] = (wb1.val & 0xc0000000) | uint32(wb1.rlen)
				//				debug.Printf("wb1 %2d %v", i, wb1)
				//				debug.Printf("res %2d %v", k, WahlBlock(res[k]))
				i++
				k++
			}
			copy(res[k:], w1.arr[i:])
			//			n := copy(res[k:], w1.arr[i:])
			//			debug.Printf("copy(res[%d:], w1.arr[%d:]) -> %d", k+1, i+1, n)
		case i >= wlen1:
			if wb2.rlen > 0 {
				res[k] = (wb2.val & 0xc0000000) | uint32(wb2.rlen)
				//				debug.Printf("wb2 %2d %v", j, wb2)
				//				debug.Printf("res %2d %v", k, WahlBlock(res[k]))
				j++
				k++
			}
			copy(res[k:], w2.arr[j:])
			//			n := copy(res[k:], w2.arr[j:])
			//			debug.Printf("copy(res[%d:], w2.arr[%d:]) -> %d", k+1, j+1, n)
		default:
			panic(errors.Bug("(i:%d of %d) - (j:%d of %d)", i, wlen1, j, wlen2))
		}
	}

	//	t0.Mark("blockwise - tail")
	/// compress results ////////////////////////////////////////////

	wahl := &Wahl{res}
	wahl.Compress()

	//	t0.Mark("blockwise - compress")
	return wahl, nil
}

func (w1 *Wahl) blockwise(op bitwiseOp, w2 *Wahl) (i, j, k int, res []uint32, wb1, wb2 wahlBlock) {

	/// recover from expected OOR error /////////////////////////////

	defer func() {
		const oor = "runtime error: index out of range"
		if x := recover(); x != nil {
			if e, ok := x.(error); ok && e.Error() == oor {
				return // ok - expected
			}
			panic(errors.Bug("unexpected: %v", x))
		}
	}()

	/// santa's little helpers //////////////////////////////////////

	var mixpair func(int, uint32) uint32
	var tilepair func(a, b uint32) uint32
	var fillpair func(a, b int) uint32
	switch op {
	case AndOp:
		mixpair = func(fval int, val uint32) uint32 {
			if fval == 1 {
				return val // val & all 1 is val
			}
			return 0 // val & all 0 is 0
		}
		tilepair = func(a, b uint32) uint32 { return a & b }
		fillpair = func(a, b int) uint32 { return uint32((a & b) << 30) }
	case OrOp:
		mixpair = func(fval int, val uint32) uint32 {
			if fval == 1 {
				return 0x7fffffff // val | all 1 is all 1
			}
			return val // val | all 0 is val
		}
		tilepair = func(a, b uint32) uint32 { return a | b }
		fillpair = func(a, b int) uint32 { return uint32((a | b) << 30) }
	case XorOp:
		mixpair = func(fval int, val uint32) uint32 {
			if fval == 1 {
				return 0x7fffffff ^ val // do xor
			}
			return val // val ^ all 0 is val
		}
		tilepair = func(a, b uint32) uint32 { return a ^ b }
		fillpair = func(a, b int) uint32 { return uint32((a ^ b) << 30) }
	}

	emit := func(info string, i, j, k int, wb1, wb2, wb3 wahlBlock) {
		//		debug.Printf("--- %s ------------------", info)
		//		debug.Printf("%2d: %v", i, wb1)
		//		debug.Printf("%2d: %v", j, wb2)
		//		debug.Printf("%2d: %v", k, wb3)
	}

	/// loop ////////////////////////////////////////////////////////

	const _TT, _TF, _FT, _FF = 0, 1, 2, 3
	var es byte

	res = make([]uint32, (maxInt(w1.Max(), w2.Max())/31)+1)
outer:
	for {
		wb1 = WahlBlock(w1.arr[i])
		wb2 = WahlBlock(w2.arr[j])
	inner:
		for {
			es = byte(wb2.val>>31 | ((wb1.val >> 31) << 1))
			switch es {
			case _TT:
				res[k] = tilepair(wb1.val, wb2.val)
				emit("tile-tile", i, j, k, wb1, wb2, WahlBlock(res[k]))
				wb1.rlen = 0
				wb2.rlen = 0
				k++
				i++
				j++
				continue outer
			case _TF:
				res[k] = mixpair(wb2.fval, wb1.val)
				emit("tile-fill", i, j, k, wb1, wb2, WahlBlock(res[k]))
				wb1.rlen = 0
				wb2.rlen--
				k++
				i++
				if wb2.rlen > 0 {
					wb1 = WahlBlock(w1.arr[i])
					continue inner
				}
				j++
				continue outer
			case _FT:
				res[k] = mixpair(wb1.fval, wb2.val)
				emit("fill-tile", i, j, k, wb1, wb2, WahlBlock(res[k]))
				wb1.rlen--
				wb2.rlen = 0
				k++
				j++
				if wb1.rlen > 0 {
					wb2 = WahlBlock(w2.arr[j])
					continue inner
				}
				i++
				continue outer
			case _FF:
				fill := uint32(0x80000000) | fillpair(wb1.fval, wb2.fval)
				switch {
				case wb1.rlen > wb2.rlen:
					res[k] = fill | uint32(wb2.rlen)
					emit("fill-fill wb1 > wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					wb1.rlen -= wb2.rlen
					wb2.rlen = 0
					k++
					j++
					wb2 = WahlBlock(w2.arr[j])
				case wb1.rlen < wb2.rlen:
					res[k] = fill | uint32(wb1.rlen)
					emit("fill-fill wb1 < wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					wb2.rlen -= wb1.rlen
					wb1.rlen = 0
					k++
					i++
					wb1 = WahlBlock(w1.arr[i])
				default:
					res[k] = fill | uint32(wb1.rlen)
					emit("fill-fill wb1 = wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					wb1.rlen = 0
					wb2.rlen = 0
					i++
					j++
					k++
					continue outer
				}
			}
		}
	}
	return
}

// Returns the position of all set bits in the bitmap. The returned
// bits are in ascending order. Returns array may be empty but never nil.
func (w *Wahl) Bits() Bitnums {
	var bits []int
	var p0 int // bit position of the initial bit in the block
	if e := w.apply(getBitsVisitor(&bits, &p0)); e != nil {
		panic(errors.Bug("Wahl.Bits: %v", e))
	}
	return Bitnums(bits)
}

// Returns the maximal bit position
func (w *Wahl) Max() int {
	// Note: since bit positions start from 0, the max register is initialized
	// a -1. However, as a necessary convention (signed vs unsigned conversions)
	// an empty zero-len Wahl bitmap will have a 'max' bit position of 0.
	var max int = -1
	if e := w.apply(maxBitsVisitor(&max)); e != nil {
		panic(errors.Bug("Wahl.Max: %v", e))
	}
	if max == -1 {
		max = 0
	}
	return max
}

// Note that bits are reversed and printed LSB -> MSB
func (w Wahl) Print(writer io.Writer) {
	var max int = -1
	if e := w.apply(printVisitor(writer, &max)); e != nil {
		panic(errors.Bug("Wahl.Print: %v", e))
	}
	fmt.Fprintf(writer, "\n")
}

/// Wahl codecs ////////////////////////////////////////////////////////////////

// REVU also New from array (buf or blocks) for (memory mapped) files.

// Writes the bitmap blocks to the given []byte slice.
// Error is returned if buf is nil or buf.len < wahl.Size().
func (w *Wahl) Encode(buf []byte) error {
	if buf == nil {
		return errors.Error("Wahl.Encode: invalid arg - buf is nil")
	}
	var wlen = len(w.arr)
	if len(buf) < (wlen << 2) {
		return errors.Error("Wahl.Encode: invalid arg - buf.len: %d", len(buf))
	}
	for i := 0; i < wlen; i++ {
		*(*uint32)(unsafe.Pointer(&buf[i<<2])) = w.arr[i]
	}
	return nil
}

// Reads 32-bit words for the bitmap blocks from the given []byte slice.
// Returns error on nil input arg.
func (w *Wahl) Decode(buf []byte) error {
	if buf == nil {
		return errors.Error("Wahl.Decode: invalid arg - buf is nil")
	}
	w.arr = make([]uint32, len(buf)>>2)
	for i := 0; i < len(w.arr); i++ {
		w.arr[i] = *(*uint32)(unsafe.Pointer(&buf[i<<2]))
	}
	return nil
}

/// Wahl visitors //////////////////////////////////////////////////////////////

// Visit function type for Wahl blocks.
type visitFn func(bn int, val uint32) (done bool, err error)

func getBitsVisitor(bits *[]int, p0 *int) visitFn {
	return func(bnum int, bval uint32) (bool, error) {
		block := WahlBlock(bval)
		if block.fill && block.fval == 1 {
			bitcnt := uint(block.rlen * 31)
			var blockBits = make([]int, bitcnt)
			for i := 0; i < len(blockBits); i++ {
				blockBits[i] = *p0 + i
			}
			*bits = append(*bits, blockBits...)
		} else if !block.fill {
			// need to check individual bits in tile
			var blockBits [31]int
			var j int // indexes blockbits
			for i := 0; i < 32; i++ {
				if block.val&0x1 == 1 {
					blockBits[j] = *p0 + i
					j++
				}
				block.val >>= 1
			}
			*bits = append(*bits, blockBits[:j]...)
		}
		*p0 += int(block.rlen * 31)
		return false, nil
	}
}

func maxBitsVisitor(max *int) visitFn {
	return func(bnum int, bval uint32) (bool, error) {
		*max += int(31 * WahlBlock(bval).rlen)
		return false, nil
	}
}

//func debugVisitor(w io.Writer, max *int) visitFn {
func printVisitor(w io.Writer, max *int) visitFn {
	return func(bnum int, bval uint32) (bool, error) {
		block := WahlBlock(bval)
		r0 := *max + 1
		*max += int(31 * block.rlen)
		fmt.Fprintf(w, "[%4d]:%s (%d, %d)\n", bnum, block, r0, *max)
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

/// Wahl helper types //////////////////////////////////////////////////////////

// Bitnums is a helper type for pretty printing bitnums
type Bitnums []int

func (a Bitnums) Print(w io.Writer) {
	fmt.Fprintf(w, "{ ")
	for _, pos := range a {
		fmt.Fprintf(w, "%d ", pos)
	}
	fmt.Fprintf(w, "}\n")
}

// wahlBlock explicitly expresses the semantics of a Wahl block uint32 value
type wahlBlock struct {
	val  uint32
	fill bool
	fval int
	rlen int // 1 for tiles assumed in bitwise() - do not change it.
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

// blockRange returns the min, max bit range of the given block 'b',
// per a prior block's maximum max0. The final return value is the
// run-length of the block b (which is independent of max0).
func blockRange(b uint32, max0 int) (uint, uint, int) {
	min := uint(max0) + 1
	n := 1
	if b>>31 > 0 {
		n = int(b & 0x3fffffff)
	}
	return min, min + (uint(n) * 31) - 1, n
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
		rlen := int(b.val & 0x3fffffff)
		return fmt.Sprintf("%030b %02b %-6s (+%-5d) *(+%-5d)",
			revbit>>2, revbit&0x3, typ, rlen*31, b.rlen*31)
	}
	return fmt.Sprintf("%031b-  %-6s +31   ", revbit>>1, typ)
}
