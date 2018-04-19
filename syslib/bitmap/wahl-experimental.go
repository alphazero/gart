// Doost

package bitmap

import (
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
)

func (w1 *Wahl) bitwise0(op bitwiseOp, w2 *Wahl) []uint32 {

	// apply logical op block by block
	// es positions are w1_w2 e.g. TF means w1.tile w2.fill
	i, j, k, res, wb1, wb2 := w1.blockwise(op, w2)

	/// op finalization /////////////////////////////////////////////

	// here we check if any partially processed tail blocks remain
	// we only care if op is XOR | OR and append those directly to blockwise result.
	if op != AndOp {
		var wlen1, wlen2 int = w1.Len(), w2.Len()
		var xarr []uint32
		var xwb wahlBlock
		var xoff int
		switch {
		case i >= wlen1 && j >= wlen2:
		case j >= wlen2:
			debug.Printf("tail @ i:%d\n", i)
			if wb1.rlen == 0 {
				wb1 = WahlBlock(w1.arr[i])
			}
			xoff = i
			xarr = w1.arr
			xwb = wb1
		case i >= wlen1:
			debug.Printf("tail @ j:%d\n", j)
			if wb2.rlen == 0 {
				wb2 = WahlBlock(w2.arr[j])
			}
			xarr = w2.arr
			xwb = wb2
			xoff = j
		default:
			panic(errors.Bug("(i:%d of %d) - (j:%d of %d)", i, wlen1, j, wlen2))
		}
		debug.Printf("--------------")
		if xarr != nil {
			// mask off the first partial block in case it was a fill block
			res[k] = (xwb.val & 0xc0000000) | uint32(xwb.rlen)
			debug.Printf("xwb %d %v", xoff, xwb)
			debug.Printf("res %d %v", k, WahlBlock(res[k]))
			k++
			xoff++
			if xoff < len(xarr) {
				n := copy(res[k:], xarr[xoff:])
				debug.Printf("copy(res[%d:], xarr[%d:]) -> %d", k, xoff, n)
			}
		}
		debug.Printf("--------------")
	}
	return res
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

	emit := func(mtyp byte, info string, i, j, k int, wb1, wb2, wb3 wahlBlock) {
		debug.Printf("--- %02b %s ------------------", byte(mtyp), info)
		debug.Printf("%2d: %v", i, wb1)
		debug.Printf("%2d: %v", j, wb2)
		debug.Printf("%2d: %v", k, wb3)
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
				emit(es, "tile-tile", i, j, k, wb1, wb2, WahlBlock(res[k]))
				wb1.rlen = 0
				wb2.rlen = 0
				k++
				i++
				j++
				continue outer
			case _TF:
				res[k] = mixpair(wb2.fval, wb1.val)
				emit(es, "tile-fill", i, j, k, wb1, wb2, WahlBlock(res[k]))
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
				emit(es, "fill-tile", i, j, k, wb1, wb2, WahlBlock(res[k]))
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
					emit(es, "fill-fill wb1 > wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					wb1.rlen -= wb2.rlen
					wb2.rlen = 0
					k++
					j++
					wb2 = WahlBlock(w2.arr[j])
				case wb1.rlen < wb2.rlen:
					res[k] = fill | uint32(wb1.rlen)
					emit(es, "fill-fill wb1 < wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
					wb2.rlen -= wb1.rlen
					wb1.rlen = 0
					k++
					i++
					wb1 = WahlBlock(w1.arr[i])
				default:
					res[k] = fill | uint32(wb1.rlen)
					emit(es, "fill-fill wb1 = wb2", i, j, k, wb1, wb2, WahlBlock(res[k]))
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
