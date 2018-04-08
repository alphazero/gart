// Doost!

package bitmap

import (
	"github.com/alphazero/gart/syslib/errors"
)

func (w Wahl) Or(other *Wahl) (*Wahl, error) {
	if other == nil {
		return nil, errors.ErrInvalidArg
	}

	// alias bitmap names for notational convenience
	var w1, w2 = w, other
	var i, j int // i indexes w1, j indexes w2
	var wlen1, wlen2 int = w1.Len(), w2.Len()
	var res = make([]uint32, (maxInt(w1.Max(), w2.Len())/31)+1)
	var k int // k indexes res
outer:
	for i < wlen1 && j < wlen2 {
		var wb1 = WahlBlock(w1.arr[i])
		var wb2 = WahlBlock(w2.arr[j])
	inner:
		for {
			switch {
			case !(wb1.fill || wb2.fill): // two tiles
				res[k] = wb1.val | wb2.val // HERE s/&/|
				k++
				i++
				j++
				continue outer
			case wb1.fill && wb2.fill: // two fills
				fill := uint32(0x80000000) | uint32((wb1.fval|wb2.fval)<<30) // HERE
				switch {
				case wb1.rlen > wb2.rlen:
					rlen := uint32(wb2.rlen)
					fill |= rlen
					res[k] = fill
					k++
					j++
					if j >= wlen2 {
						continue outer // just a formality - it's done
					}
					wb1.rlen -= wb2.rlen
					wb2 = WahlBlock(w2.arr[j])
					continue inner
				case wb1.rlen < wb2.rlen:
					rlen := uint32(wb1.rlen)
					fill |= rlen
					res[k] = fill
					k++
					i++
					if i >= wlen1 {
						continue outer // just a formality - it's done
					}
					wb2.rlen -= wb1.rlen
					wb1 = WahlBlock(w1.arr[i])
					continue inner
				default: // a match made in heaven ..
					rlen := uint32(wb1.rlen)
					fill |= rlen
					res[k] = fill
					k++
					i++
					j++
					continue outer
				}
			case wb1.fill: // w2 is a tile
				var tile = wb2.val // assume tile 0 HERE
				if wb1.fval == 1 {
					tile = 0x7FFFFFFF
				}
				res[k] = tile
				k++
				j++
				if j >= wlen2 {
					continue outer // just a formality - it's done
				}
				wb2 = WahlBlock(w2.arr[j])
				if wb1.rlen > 1 {
					wb1.rlen--
					continue inner
				}
				i++
				if i >= wlen1 {
					continue outer // just a formality - it's done
				}
				continue outer
			case wb2.fill:
				var tile = wb2.val // assume tile 0 HERE
				if wb1.fval == 1 {
					tile = 0x7FFFFFFF
				}
				res[k] = tile
				k++
				i++
				if i >= wlen1 {
					continue outer // just a formality - it's done
				}
				wb1 = WahlBlock(w1.arr[i])
				if wb2.rlen > 1 {
					wb2.rlen--
					continue inner
				}
				j++
				if j >= wlen2 {
					continue outer // just a formality - it's done
				}
				continue outer
			}
		}
	}
	wahl := &Wahl{res}
	wahl.Compress()

	return wahl, nil
}
