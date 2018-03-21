// Doost!
package sort_test

import (
	"testing"

	"github.com/alphazero/gart/syslib/sort"
)

func TestUint64Slice(t *testing.T) {
	var arr = []uint64{34, 57, 17, 18, 27}
	var exp = []uint64{17, 18, 27, 34, 57}
	sort.Uint64(arr)
	for i := 0; i < len(arr); i++ {
		if arr[i] != exp[i] {
			t.Errorf("have: %d - expect: %d", arr[i], exp[i])
		}
	}
}
