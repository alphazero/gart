// Doost!

package test

import (
	"fmt"

	"github.com/alphazero/gart/syslib/bitmap"
	"github.com/alphazero/gart/syslib/errors"
)

//var _ = debug.Printf
type opdef struct {
	fn   func(...*bitmap.Wahl) (*bitmap.Wahl, error)
	name string
}

var bitwiseOps = []*opdef{
	&opdef{bitmap.And, "AND"},
	&opdef{bitmap.Or, "OR "},
	&opdef{bitmap.Xor, "XOR"},
}

func verifySet(w *bitmap.Wahl, a []uint) {
	w_map := mapArray(w.Bits())
	for _, bit := range a {
		// bit must be in map
		if !w_map[int(bit)] {
			panic(errors.Bug("Set: bit %d is not in bitmap\n", bit))
		}
	}
}

func verifyClear(w *bitmap.Wahl, a []uint) {
	w_map := mapArray(w.Bits())
	for _, bit := range a {
		// bit must -not- be in map
		if w_map[int(bit)] {
			panic(errors.Bug("Clear: bit %d is in bitmap\n", bit))
		}
	}
}

func verifyNot(w, wnot *bitmap.Wahl) error {
	var w_max, wnot_max = w.Max(), wnot.Max()
	if w_max != wnot_max {
		panic(errors.Bug("NOT: max %d != !max %d\n", w_max, wnot_max))
	}
	w_map := mapArray(w.Bits())
	wnot_map := mapArray(wnot.Bits())
	for n := 0; n < w_max; n++ {
		// if n is in w it must -not- be in wnot
		// if n is in wnot it must -not- be in w
		if w_map[n] && wnot_map[n] {
			return errors.Bug("NOT: bit %d is set in both\n", n)
		}
	}
	return nil
}

// and.Max must be equal to min(a.Max, b.Max)
func verifyAnd(a, b, and *bitmap.Wahl) error {
	a_map := mapArray(a.Bits())
	b_map := mapArray(b.Bits())
	and_map := mapArray(and.Bits())
	ref_map := andMaps(a_map, b_map)
	compareMaps("verify AND map", and_map, ref_map)
	for _, bit := range and.Bits() {
		// bit must be in both maps for AND
		if !(a_map[bit] && b_map[bit]) {
			return errors.Bug("AND: bit %d is not set in both maps\n", bit)
		}
	}
	return nil
}

// or.Max must be equal to max(a.Max, b.Max)
func verifyOr(a, b, or *bitmap.Wahl) error {
	a_map := mapArray(a.Bits())
	b_map := mapArray(b.Bits())
	or_map := mapArray(or.Bits())
	ref_map := orMaps(a_map, b_map)
	compareMaps("verify OR map", or_map, ref_map)
	ab_max := max(a.Max(), b.Max())
	if or.Max() != ab_max {
		return errors.Bug("OR: tail-error - Max(a:%d, b:%d) and or.Max:%d\n",
			a.Max(), b.Max(), or.Max())
	}
	for _, bit := range or.Bits() {
		// bit must be in one or both of maps for or
		if !(a_map[bit] || b_map[bit]) {
			return errors.Bug("OR: bit %d is not set in either maps\n", bit)
		}
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// BUG not yet done.
// xor.Max must be equal to max(a.Max, b.Max)
func verifyXor(a, b, xor *bitmap.Wahl) error {
	a_map := mapArray(a.Bits())
	b_map := mapArray(b.Bits())
	xor_map := mapArray(xor.Bits())
	ref_map := xorMaps(a_map, b_map)
	compareMaps("verify XOR map", xor_map, ref_map)
	ab_max := max(a.Max(), b.Max())
	if xor.Max() != ab_max {
		return errors.Bug("OR: tail-error - Max(a:%d, b:%d) and or.Max:%d\n",
			a.Max(), b.Max(), xor.Max())
	}
	for _, bit := range xor.Bits() {
		a_val := boolToInt(a_map[bit])
		b_val := boolToInt(b_map[bit])
		xor_val := a_val ^ b_val
		if xor_val != 1 {
			return errors.Bug("XOR: bit %d is not XOR of a:%d b:%d \n", bit, a_val, b_val)
		}
	}
	return nil
}

// asserts maps are identical: have same length and same content
func compareMaps(info string, a, b map[int]bool) {
	if len(a) != len(b) {
		panic(errors.Bug("%s - a.len:%d != b.len:%d", info, len(a), len(b)))
	}
	for k, v := range a {
		if _, ok := b[k]; !ok {
			panic(fmt.Sprintf("%s - k:%d (%t)\n", info, k, v))
		}
	}
}

func smallerFirst(a, b []int) ([]int, []int) {
	if len(a) < len(b) {
		return a, b
	}
	return b, a
}

func xorMaps(a, b map[int]bool) map[int]bool {
	xor := make(map[int]bool)
	var a_max, b_max int

	for k, _ := range a {
		if !b[k] {
			//			fmt.Printf("XOR-debug {A}: a[%d]=%t b[%d]=%t\n", k, a[k], k, b[k])
			xor[k] = true
		}
		if k > a_max {
			a_max = k
		}
	}
	for k, _ := range b {
		if !a[k] {
			//			fmt.Printf("XOR-debug {B}: b[%d]=%t a[%d]=%t\n", k, b[k], k, a[k])
			xor[k] = true
		}
		if k > b_max {
			b_max = k
		}
	}
	return xor
}

func orMaps(a, b map[int]bool) map[int]bool {
	or := make(map[int]bool)
	for k, _ := range a {
		or[k] = true
	}
	for k, _ := range b {
		or[k] = true
	}
	return or
}

func andMaps(a, b map[int]bool) map[int]bool {
	and := make(map[int]bool)
	for k, _ := range a {
		if t, ok := b[k]; ok && t {
			and[k] = true
		}
	}
	return and
}

func mapArray(a []int) map[int]bool {
	a_map := make(map[int]bool)
	for _, v := range a {
		a_map[v] = true
	}
	return a_map
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
