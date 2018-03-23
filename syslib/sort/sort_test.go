// Doost!
package sort_test

import (
	"testing"

	"github.com/alphazero/gart/syslib/sort"
)

func TestUintSlice(t *testing.T) {
	var arr = []uint{34, 57, 17, 18, 27}
	var exp = []uint{17, 18, 27, 34, 57}
	sort.Uints(arr)
	for i := 0; i < len(arr); i++ {
		if arr[i] != exp[i] {
			t.Errorf("have: %d - expect: %d", arr[i], exp[i])
		}
	}
}

func TestUint64Slice(t *testing.T) {
	var arr = []uint64{34, 57, 17, 18, 27}
	var exp = []uint64{17, 18, 27, 34, 57}
	sort.Uint64s(arr)
	for i := 0; i < len(arr); i++ {
		if arr[i] != exp[i] {
			t.Errorf("have: %d - expect: %d", arr[i], exp[i])
		}
	}
}
