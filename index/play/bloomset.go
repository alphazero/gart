// Doost!

package main

import (
	"flag"
	"fmt"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/sort"
	"math/rand"
	"time"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")
	flag.Parse()
	run()
}

var option = struct {
	n int
	d int
	B int
}{
	n: 1000,
	d: 0x7fff,
	B: 8,
}

func init() {
	flag.IntVar(&option.B, "B", option.B, "bucket depth")
	flag.IntVar(&option.n, "n", option.n, "number of elements")
	flag.IntVar(&option.d, "d", option.d, "distance between elements")
}

func run() {

	var set = NewSet64(option.n)

	//	var arr = make([]uint64, n)
	for i := 0; i < option.n; i++ {
		s := fmt.Sprintf("this is a fix prefix %b this is a fixed postfix", i)
		//		s = fmt.Sprintf("%b %v ", i, 0) //time.Now())
		//		h := digest.SumUint64s([]byte(s))
		//		arr[i] = h[0] & mask
		set.Put([]byte(s))
		//		fmt.Printf("%d\n", i)
	}
	println("loaded")
	//	var set = BuildSet64(arr)
	set.Sort()
	println("sorted")

	if true {
		dtimes = 0
		ntimes = 0
		for i, x := range set.arr {
			if !set.find(x) {
				fmt.Printf("not found %d\n", x)
				fmt.Printf("was actually here %d arr[j]:%d\n", i, set.arr[i])
				panic("bug")
			}
		}
		fmt.Printf("%f ns/successful find\n", float64(dtimes)/float64(ntimes))
	}
	var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		dtimes = 0
		ntimes = 0

		const iters = 1000
		for i := 0; i < iters; i++ {
			s := fmt.Sprintf("%v", i*rnd.Int())
			if set.Contains([]byte(s)) {
				fmt.Printf("xxx found %d\n", i)
			}
		}
		fmt.Printf("%f ns/unsuccessful find\n", float64(dtimes)/float64(ntimes))
	}
}

var ntimes int
var dtimes int64

/// types //////////////////////////////////////////////////////////////////////

// REVU this type is problematic but is a good proof of concept for 'compressed
// bloom filter' approach. This is effectively run-length encoding 'bits' in a
// space of 'u'.
//
// This is really a static type. We spec a capacity n and are expected to fill
// the set with n items. (REVU what happens if it does not?)
//
// Then before query, we must 'build' it (e.g. sort it).
//
// Then we can use it.
type Set64 struct {
	arr  []uint64
	K    int
	k    float64
	u    uint64
	mask uint64
	n    int
}

// REVU this is conceptually broken.
func BuildSet64(arr []uint64) *Set64 {
	sort.Uint64s(arr)
	s := &Set64{}
	s.arr = arr
	s.u = uint64(1 << 63)
	s.k = float64(len(arr)) / float64(s.u)

	return s
}

func NewSet64(n int) *Set64 {
	var K = 2
	var u = uint64(1 << 63)
	return &Set64{
		arr:  make([]uint64, n*K),
		u:    u,
		mask: u - 1,
		k:    float64(n*K) / float64(1<<63),
		K:    K,
	}
}

func (s *Set64) Put(b []byte) {
	h := digest.SumUint64s(b)
	for j := 0; j < s.K; j++ {
		s.arr[s.n] = h[j] & (s.mask)
		s.n++
	}
}

func (s *Set64) Sort() {
	sort.Uint64s(s.arr)
}

func (s *Set64) Contains(b []byte) bool {
	h := digest.SumUint64s(b)
	var ok = true
	for j := 0; j < s.K; j++ {
		ok = ok && s.find(h[j]&(s.mask))
	}
	return ok
}

// Interpolation based search. Is faster than binary search when bigger than cache.
func (s *Set64) find(v uint64) bool {
	var start = time.Now().UnixNano()
	defer func() {
		dtimes += (time.Now().UnixNano() - start)
		ntimes++
	}()
	if v >= s.u {
		return false
	}
	//	fmt.Println("                                                                |     ")
	var i = int(float64(v) * s.k)
	if s.arr[i] < v {
		dx := int(float64(v-s.arr[i])*s.k) >> 1
		for i < len(s.arr)-1 && s.arr[i] < v {
			//			fmt.Printf(">")
			if dx == 0 || i+dx >= len(s.arr) {
				break
			}
			i += dx
			dx = (dx >> 1)
		}
	} else if s.arr[i] > v {
		dx := int(float64(s.arr[i]-v)*s.k) >> 1
		for i >= 0 && s.arr[i] > v {
			//			fmt.Printf("<(%d, %d) ", i, dx)
			if dx == 0 || dx > i {
				break
			}
			i -= dx
			dx = dx >> 1
		}
	}
	if i < 0 || i >= len(s.arr) {
		// fmt.Println()
		return false
	}
	for i > 0 && s.arr[i] > v {
		//		fmt.Printf("-")
		i--
	}
	for i < len(s.arr)-1 && s.arr[i] < v {
		//		fmt.Printf("+")
		i++
	}
	//	fmt.Println()
	return s.arr[i] == v
}
