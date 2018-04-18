// Doost

package bitmap

import (
	"fmt"
	"github.com/alphazero/gart/syslib/errors"
)

func (w1 *Wahl) bitwise0(op bitwiseOp, w2 *Wahl) []uint32 {
	i, j, k, res, wb1, wb2, es := w1.blockwise(op, w2)

	var wlen1, wlen2 int = w1.Len(), w2.Len()
	/// op finalization /////////////////////////////////////////////

	// here let's check if any partially processed tail blocks remain
	// we only care if op is XOR | OR.
	// If loop terminated in end-state where the last block of the incompletely
	// processed bitmap was a tile, then its associated wahlBlock (wb#) is pointing
	if op != AndOp {
		var xarr []uint32
		var xwb wahlBlock
		var xoff int
		switch {
		case i >= wlen1 && j >= wlen2:
		case j >= wlen2:
			fmt.Printf("i:%d - end-state:%s\n", i, es)
			if es[0] == 'T' {
				wb1 = WahlBlock(w1.arr[i])
			}
			xoff = i
			xarr = w1.arr
			xwb = wb1
		case i >= wlen1:
			fmt.Printf("j:%d - end-state:%s\n", j, es)
			if es[2] == 'T' {
				wb2 = WahlBlock(w2.arr[j])
			}
			xarr = w2.arr
			xwb = wb2
			xoff = j
		default:
			panic(errors.Bug("(i:%d of %d) - (j:%d of %d)", i, wlen1, j, wlen2))
		}
		fmt.Println("--------------")
		if xarr != nil {
			// mask off the first partial block in case it was a fill block
			res[k] = (xwb.val & 0xc0000000) | uint32(xwb.rlen)
			fmt.Printf("xwb %d %v\n", xoff, xwb)
			fmt.Printf("res %d %v\n", k, WahlBlock(res[k]))
			k++
			xoff++
			if xoff < len(xarr) {
				n := copy(res[k:], xarr[xoff:])
				fmt.Printf("copy(res[%d:], xarr[%d:]) -> %d\n", k, xoff, n)
			}
		}
		fmt.Println("--------------")
	}
	return res
}

func (w1 *Wahl) blockwise(op bitwiseOp, w2 *Wahl) (i, j, k int, res []uint32, wb1, wb2 wahlBlock, es string) {

	/// recover from expected OOR error /////////////////////////////

	defer func() {
		const oor = "runtime error: index out of range"
		if x := recover(); x != nil {
			if e, ok := x.(error); ok && e.Error() == oor {
				fmt.Printf("%q\n", e)
				return
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

	emit := func(mtyp byte, info string, i, j, k int, wb1, wb2, wb3 wahlBlock) {
		fmt.Printf("--- %02b %s ------------------ \n", byte(mtyp), info)
		fmt.Printf("%2d: %v\n", i, wb1)
		fmt.Printf("%2d: %v\n", j, wb2)
		fmt.Printf("%2d: %v\n", k, wb3)
	}

	/// loop ////////////////////////////////////////////////////////

	const TT, TF, FT, FF = 0x00, 0x01, 0x10, 0x11
	res = make([]uint32, (maxInt(w1.Max(), w2.Max())/31)+1)
outer:
	for {
		wb1 = WahlBlock(w1.arr[i])
		wb2 = WahlBlock(w2.arr[j])
		/*
			00 TT
			01 TF
			10 FT
			11 FF
		*/
	inner:
		for {
			mtyp := byte(wb2.val>>31 | ((wb1.val >> 31) << 1))
			switch {
			//			switch mtyp {
			case !(wb1.fill || wb2.fill): // two tiles
				//	case TT:
				es = "T-T"
				res[k] = tilepair(wb1.val, wb2.val)
				emit(mtyp, "tile-tile", i, j, k, wb1, wb2, WahlBlock(res[k]))
				k++
				i++
				j++
				continue outer
			case wb1.fill && wb2.fill: // two fills
				//			case FF:
				es = "F-F"
				fill := uint32(0x80000000) | fillpair(wb1.fval, wb2.fval)
				switch {
				case wb1.rlen > wb2.rlen:
					wb1.rlen -= wb2.rlen
					rlen := uint32(wb2.rlen)
					fill |= rlen
					res[k] = fill
					emit(mtyp, "fill-fill wb1 > wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					k++
					j++
					wb2 = WahlBlock(w2.arr[j])
				case wb1.rlen < wb2.rlen:
					wb2.rlen -= wb1.rlen
					rlen := uint32(wb1.rlen)
					fill |= rlen
					res[k] = fill
					emit(mtyp, "fill-fill wb1 < wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					k++
					i++
					wb1 = WahlBlock(w1.arr[i])
				default: // a match made in heaven ..
					rlen := uint32(wb1.rlen)
					fill |= rlen
					res[k] = fill
					emit(mtyp, "fill-fill wb1 = wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					i++
					j++
					k++
					continue outer
				}
			case wb1.fill: // w2 is a tile
				//	case FT:
				es = "F-T"
				res[k] = mixpair(wb1.fval, wb2.val)
				wb1.rlen--
				emit(mtyp, "fill-tile", i, j, k, wb1, wb2, WahlBlock(res[k]))
				k++
				j++
				if wb1.rlen > 0 {
					wb2 = WahlBlock(w2.arr[j])
					continue inner
				}
				i++
				continue outer
			case wb2.fill: // w1 is a tile
				//	case TF:
				es = "T-F"
				res[k] = mixpair(wb2.fval, wb1.val)
				wb2.rlen--
				emit(mtyp, "tile-fill", i, j, k, wb1, wb2, WahlBlock(res[k]))
				k++
				i++
				if wb2.rlen > 0 {
					wb1 = WahlBlock(w1.arr[i])
					continue inner
				}
				j++
				continue outer
			}
		}
	}
	return
}
