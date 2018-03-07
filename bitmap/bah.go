package bitmap

import (
	"fmt"
)

// BAH07 compression is a minor variation on WAH bitmap compression, the
// difference being:
//
//	- BAH07 is Byte aligned (whereas WAH is 32bit Word aligned)
//	- BAH07 only compresses consecutive 0s so there is no fill-pattern bit
//  - BAH07 has a 7bit run length multiples of 8 0s (max 127x8=1026 consecutive 0s)
//
//  For example, the following 8 byte 7-encoded bitmap
//
//      0101101 0000000 0000000 0000000 0000000 0000000 0000000 00001000
//
//  is BAH07 compressed to 3 bytes
//
//		+-+-+-+-+-+-+-+  +-+-+-+-+-+-+-+  +-+-+-+-+-+-+-+
//		|0|1|0|1|1|0|1|  |1|0|0|0|1|0|1|  |0|0|0|0|1|0|0|
//		+-+-+-+-+-+-+-+  +-+-+-+-+-+-+-+  +-+-+-+-+-+-+-+
//      [0]              [1]              [2]
//
//  Note: it is important to keep in mind that BAH07 compression is specifically designed
//        to work with tag ids, which are all guaranteed to be relatively prime to 8, and
//        that is the reason rulen is multiplied by 8. In effect, the MSB control bit is
//        actually the bit position of ids that can never occur. This is NOT a general
//        purpose compression library.

var _ = fmt.Printf

// used for decompression
var fillPattern [127]byte

/// compression - decompression ///////////////////////////////////////////////

// Compress expects a 7bit-encoded array, meaning each element of in-arg bitmap
// is expected to have 0 at its MSB. Contrary input is treated as bug.
//
// Returns a BAH07 compressed bitmap. In-arg is not modified.
// Function panics if input is not 7bit-encoded.
func compress(bitmap []byte) []byte {
	var nopmap [1]byte
	var ulen = len(bitmap)

	// trivial cases
	if ulen == 0 {
		return []byte{}
	} else if ulen == 1 {
		nopmap[0] = bitmap[0]
		return nopmap[:]
	}

	// we have a bitmap of at least 2 bytes, so we can meaningfully
	// compress it.
	var i int
	var compressed []byte
	for i < ulen {
		var b = bitmap[i]
		switch {
		case b == 0x00:
			n, fill := fillBAH07(bitmap, i+1)
			compressed = append(compressed, fill)
			i += n
		case b&0x80 == 0x80:
			panic(fmt.Errorf("bug - bitmap.Compress: b[%d]:%08b is invalid", i, b))
		default:
			compressed = append(compressed, b)
			i++
		}
	}
	return compressed
}

// Function is called when first instance of a 1/0 filled block is encountered.
// Returns number of uncompressed blocks covered by filler (n) and the filler block.
func fillBAH07(bitmap []byte, i int) (n int, fill byte) {
	n = 1
	for _, b := range bitmap[i:] {
		switch b {
		case 0x00:
			if n == 127 {
				goto makefill
			}
			n++
		default:
			goto makefill
		}
	}

makefill:
	fill = 0x80 | byte(n)
	return
}

// Returns the decompressed psuedo 7bit encoded array.
// If in-arg is already decompressed the result will be identical.
// The input arg is not modified.
func decompress(bah []byte) []byte {
	var nopmap [1]byte
	var bahlen = len(bah)

	// trivial cases
	if bahlen == 0 {
		return []byte{}
	} else if bahlen == 1 {
		nopmap[0] = bah[0]
		return nopmap[:]
	}

	var bitmap []byte
	for i := 0; i < bahlen; i++ {
		if b := bah[i]; b&0x80 == 0 {
			bitmap = append(bitmap, b)
		} else {
			runlen := int(b & 0x7f)
			bitmap = append(bitmap, fillPattern[:runlen]...)
		}
	}
	return bitmap
}

// allSet returns true if all the in-arg bits are set in the
// in-arg bitmap. Input arg is expected to be a 7bit encoded
// array as the MSB is treated as the BAH control bit.
// Passing a decompressed array is fine but is a waste of cpu time.
//
// Note: bits must be in ascending sort order.
func allSet(bitmap []byte, bits ...int) bool {
	bitslen := len(bits)

	if bitslen == 0 {
		return false
	}

	var bit_0 int // initial bit covered by the byte block
	var bit_n int // last bit covered by the byte block
	var i int     // indexes bits - bits[i] always > bit_0

	//	fmt.Printf("DEBUG - bitmap: %08b\n", bitmap)
	for bn, b := range bitmap {
		fmt.Printf("\t\t\tremaining bits to check : %d\n", bits[i:])
		fmt.Printf("block:%d  b:%08b ", bn, b)
		switch b & 0x80 {
		case 0x80: // FILL
			runlen := int(b&0x7f) << 3 // each FILL covers 8 bits
			bit_n = bit_0 + runlen     // - 1
			fmt.Printf(" FILL [%d, %d) runlen:%d\n", bit_0, bit_n, runlen)
			for i < bitslen {
				if bits[i] < bit_n {
					return false // bit is in 0-Fill [bit_0, bit_n]
				}
				break
			}
		default:
			bit_n = bit_0 + 8
			fmt.Printf(" FORM [%d, %d)\n", bit_0, bit_n)
			for i < bitslen {
				bit := bits[i]
				if bit >= bit_n {
					break // bit is beyond this byte's range
				}
				bitmask := byte(0x80 >> uint(bit&0x7))
				if b&bitmask == 0 {
					return false
				}
				i++
			}
		}
		if i == bitslen {
			fmt.Printf("end-loop break i:%d bitslen:%d\n", i, bitslen)
			break // or just return true
		}
		bit_0 = bit_n
	}
	return true
}

// Return the corresponding boolean value of the in-arg bit posiitons.
// TODO decide regarding the bit positions outside of the range of the bitmap.
//
// Note: bits must be in ascending sort order.
// TODO TEST!
func getBitvals(bitmap []byte, bits ...int) []bool {
	bitslen := len(bits)

	if bitslen == 0 {
		return []bool{}
	}

	var bitvals = make([]bool, bitslen) // results
	var bit_0 int                       // initial bit covered by the byte block
	var bit_n int                       // last bit covered by the byte block
	var i int                           // indexes bits & bitvals - bits[i] always > bit_0

	//	fmt.Printf("DEBUG - bitmap: %08b\n", bitmap)
	for bn, b := range bitmap {
		fmt.Printf("\t\t\tremaining bits to check : %d\n", bits[i:])
		fmt.Printf("block:%d  b:%08b ", bn, b)
		switch b & 0x80 {
		case 0x80: // FILL
			runlen := int(b&0x7f) << 3 // each FILL covers 8 bits
			bit_n = bit_0 + runlen     // - 1
			fmt.Printf(" FILL [%d, %d) runlen:%d\n", bit_0, bit_n, runlen)
			for i < bitslen {
				if bits[i] >= bit_n {
					break // bit is beyond this byte's range
				}
				bitvals[i] = false // 'application'
				i++
			}
		default:
			bit_n = bit_0 + 8
			fmt.Printf(" FORM [%d, %d)\n", bit_0, bit_n)
			for i < bitslen {
				bit := bits[i]
				if bit >= bit_n {
					break // bit is beyond this byte's range
				}
				bitmask := byte(0x80 >> uint(bit&0x7))
				bitvals[i] = b&bitmask != 0
				i++
			}
		}
		if i == bitslen {
			fmt.Printf("end-loop break i:%d bitslen:%d\n", i, bitslen)
			break // or just return true
		}
		bit_0 = bit_n
	}
	return bitvals
}

// Returns true if any of the bit values are set.
// REVU this impl. is not the most efficient as getBitvals traverses  the entire
// bits array, where as (like in allSet) we simply want to return as soon as we
// hit a bit that is set.
// REVU just dup & modify the code from getBitsedicate func in-arg.
// TODO test!
func anySet(bitmap []byte, bits ...int) bool {
	bitvals := getBitvals(bitmap, bits...)
	for _, b := range bitvals {
		if b {
			return true
		}
	}
	return false
}

// REVU this has to iterate over the full bits...
func noneSet(bitmap []byte, bits ...int) bool {
	bitvals := getBitvals(bitmap, bits...)
	fmt.Printf("DEBUG - bits  s: %d\n", bits)
	fmt.Printf("DEBUG - bitvals: %t\n", bitvals)
	for i, b := range bitvals {
		if b {
			println(i)
			return false
		}
	}
	return true
}

// XXX
// REVU this really should be an object (since it needs state)
// and then the if/then statements in allSet/getBitvals are actually
// in that object. The iterator simply walks the compressed array and
// switched on block type. So we basically save a loop and switch
// code de-dup at the cost of possible inefficiencies and less strightforward
// code.
type application func(b bool) (halt bool, result interface{})

func apply(b []byte, fn application) interface{} {
	// REVU it just gets too complicated.
	//      consider AllSet, for example.
	//      also it needs an accum. of some sort.
	//      Go is not the right language for this. Here is one for FP.
	panic("not a good idea :)")
}

// XXX
