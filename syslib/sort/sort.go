// Doost!

package sort

import (
	stdlib "sort"
)

// shamelessly copied from the std-lib sort and modified for our
// favored types.

/// int64 /////////////////////////////////////////////////////////////////////

type Int64Slice []int64

func Int64s(a []int64) { stdlib.Sort(Int64Slice(a)) }

func (p Int64Slice) Len() int           { return len(p) }
func (p Int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Int64Slice) Sort()              { stdlib.Sort(p) }

/// uint64 /////////////////////////////////////////////////////////////////////

type Uint64Slice []uint64

func Uint64s(a []uint64) { stdlib.Sort(Uint64Slice(a)) }

func (p Uint64Slice) Len() int           { return len(p) }
func (p Uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Uint64Slice) Sort()              { stdlib.Sort(p) }

/// uint ///////////////////////////////////////////////////////////////////////

type UintSlice []uint

func Uints(a []uint) { stdlib.Sort(UintSlice(a)) }

func (p UintSlice) Len() int           { return len(p) }
func (p UintSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p UintSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p UintSlice) Sort()              { stdlib.Sort(p) }
