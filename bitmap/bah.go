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

// expects a 7bit segmented array
// Compress expects a 7bit-encoded array, meaning each element of in-arg bitmap
// is expected to have 0 at its MSB. Contrary input is treated as bug.
//
// Returns a BAH07 compressed bitmap. In-arg is not modified.
// Function panics if input is not 7bit-encoded.
func Compress(bitmap []byte) []byte {
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

// returns a 7bit segmented slice. In-arg is not modified.
func Decompress(bah []byte) []byte {
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

// bah.BitsSet
func BitsSet(compressed []byte, bits ...int) bool {
	bitslen := len(bits)

	if bitslen == 0 {
		return false
	}

	// for each byte in the compressed bitmap, check if any bit
	// in the block range are set.

	var bit_0 int // initial bit covered by the byte block
	var bit_n int // last bit covered by the byte block
	var i int     // indexes bits - bits[i] always > bit_0
	fmt.Printf("DEBUG - compressed: %08b\n", compressed)
	//next_byte:
	for bn, b := range compressed {
		fmt.Printf("block:%d  b:%08b ", bn, b)
		switch b & 0x80 {
		case 0x80: // FILL
			runlen := int(b&0x7f) << 3 // each FILL covers 8 bits
			bit_n = bit_0 + runlen - 1
			fmt.Printf(" FILL runlen:%d (%d, %d)\n", runlen, bit_0, bit_n)
			for i < bitslen {
				// if any bit is in 0-Fill [bit_0, bit_n] return false
				if bits[i] <= bit_n {
					//					fmt.Printf("return F b:%08b _0:%d _n:%d bits[%d]:%d\n", b, bit_0, bit_n, i, bits[i])
					return false
				}
				//				fmt.Printf("continue b:%08b _0:%d _n:%d bits[%d]:%d\n", b, bit_0, bit_n, i, bits[i])
				break
				//				i++
			}
			//			bit_0 = bit_n + 1
		default:
			bit_n = bit_0 + 7
			fmt.Printf(" FORM (%d, %d)\n", bit_0, bit_n)
			for i < bitslen {
				bit := bits[i]
				// if bit is beyond this byte's range
				if bit > bit_n {
					//					fmt.Printf("FORM break bit:%d bit_n:%d\n", bit, bit_n)
					break //continue next_byte
				}
				bitmask := byte(0x80 >> uint(bit&0x7))
				if b&bitmask == 0 {
					//					fmt.Printf("return F b:%08b _0:%d _n:%d bits[%d]:%d\n", b, bit_0, bit_n, i, bits[i])
					return false
				}
				//				fmt.Printf("continue b:%08b _0:%d _n:%d bits[%d]:%d\n", b, bit_0, bit_n, i, bits[i])
				i++
			}
		}
		if i == bitslen {
			fmt.Printf("end-loop break i:%d bitslen:%d\n", i, bitslen)
			break // or just return true
		}
		bit_0 = bit_n + 1
	}
	return true
}

func SelectsAll(bitmap []byte, n ...int) bool {
	var nlen = len(n)
	if nlen == 0 {
		return false
	}
	// NOTE from is progressively updated. To depends on block type
	var nidx int // indexes in-arg n // REVU var from int // remember, compressed ..
next_block:
	for bn, block := range bitmap {
		from := (bn << 3) - bn // REVU this can't be right either. see above
		switch block & 0x80 {  // add case for & 0xC0& panic on 11000000
		case 0x80:
			fill := (block & 0x40) >> 6 // REVU not necessary
			if fill == 0 {              // REVU this can't be right ..
				return false // ..  REVU this is wrong.
			}
			rlen := int(block & 0x3f)
			to := from + (7 * rlen)
			for nidx < nlen {
				if n[nidx] >= to {
					continue next_block
				} // REVU else { bit is in 0-fill block so return }
				nidx++
			}
		default:
			to := from + 7
			for nidx < nlen {
				bitnum := n[nidx]
				if bitnum >= to {
					continue next_block
				}
				shift := uint(to - bitnum - 1) // REVU shift := 0x80 >> (bitnum & 0x7)
				v := (block >> shift) & 0x01   // REVU block & tab[bitnum & 0x7] // ?
				if v == 0 {
					return false
				}
				nidx++
			}
		}
		if nidx == nlen {
			break
		}
	}
	if nidx < nlen {
		return false
	}
	return true
}

/*
/// bit select ops ////////////////////////////////////////////////////////////

// REVU this is fine but it is likely much faster to
// modify GetBits and pass an 'bitop' func.
func Selects(bitmap []byte, n ...int) bool {
	bitvals, oob := GetBits(bitmap, n...)
	if len(oob) > 0 {
		return false
	}
	for _, b := range bitvals {
		if !b {
			return false
		}
	}
	return true
}

// GetBits returns selected bits of compressed bitmap, as array of bool,
// corresponding to the n ...arg. NOTE that n[]  must be in ascending
// sort order. For example, if n = {1, 3, 99} and bitval[] = {true, false, true},
// then the mapping is {1->true, 3->false, 99->true}.
//
// Note thta function will not check if input is in fact sorted, and this is
// delegated to the call site.
//
// TODO numbering the bits [1, n] or [0, n) ?
//
// If any bit number exceeds the length of the decompressed bitmap, it will be returned
// in the returned out-of-bounds 'oob' array. Per above example, if the decompressed
// bitmap has only 64 bits, then results will be bitval[]={true, false}, and
// oob[]={99}.
//
// Function will never return nil values for either bitval[] or oob[].
//
// Function will panic if bitmap[] arg is nil.
func GetBits(bitmap []byte, n ...int) (bitval []bool, oob []int) {
	var nlen = len(n)
	if nlen == 0 {
		return
	}

	var boolVal = [2]bool{false, true}
	var nidx int // indexes in-arg n
next_block:
	for bn, block := range bitmap {
		from := (bn << 3) - bn
		switch block & 0x80 {
		case 0x80:
			fill := (block & 0x40) >> 6
			rlen := int(block & 0x3f)
			to := from + (7 * rlen)
			for nidx < nlen {
				//				bitnum := n[nidx]
				if n[nidx] >= to {
					continue next_block
				}
				bitval = append(bitval, boolVal[fill])
				nidx++
			}
		default:
			to := from + 7
			for nidx < nlen {
				bitnum := n[nidx]
				if bitnum >= to {
					continue next_block
				}
				shift := uint(to - bitnum - 1)
				v := (block >> shift) & 0x01
				bitval = append(bitval, boolVal[v]) // REVU: here we perform the logical op
				nidx++
			}
		}
		if nidx == nlen {
			break
		}
	}
	if nidx < nlen {
		oob = n[nidx:]
	}
	return
}

*/
