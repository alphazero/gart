// Doost!

package sort

import (
	stdlib "sort"
)

// shamelessly copied from the std-lib sort and modified for our
// favored types.

func Uint64(a []uint64) { stdlib.Sort(Uint64Slice(a)) }

type Uint64Slice []uint64

func (p Uint64Slice) Len() int           { return len(p) }
func (p Uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p Uint64Slice) Sort() { stdlib.Sort(p) }
