// Doost!

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/system/log"
)

//const size = 1 << 16

//const mask = size - 1
//const dmax = 128 // size >> 7
var d uint = 12
var h int = 4
var dmin int = 256

func init() {
	flag.UintVar(&d, "d", d, "d (degree) determines total capacity at 2^d or (1 << d)")
	flag.IntVar(&h, "h", h, "number of hash functions - min 1")
	flag.IntVar(&dmin, "p", dmin, "minimum number of probes")
}

func main() {
	flag.Parse()
	log.Verbose(os.Stdout)
	log.Log("Salaam Samad Sultan of LOVE!")

	linearProbe()
}

func linearProbe() {
	var size = 1 << d
	if h < 1 || h > 8 {
		exitOnError("h must be in range (1, 8) inclusive.")
	}
	var t [][]uint64 = make([][]uint64, h)
	var size0 = size / h //len(t)
	var mask = uint64(size0 - 1)
	var dmax = max(uint64(size0/100), uint64(dmin))
	log.Log("using %d H & %d sized segments for total capacity of %d", h, size0, size)
	for x := 0; x < h; x++ {
		t[x] = make([]uint64, size0)
	}

	var i int
next:
	for i < size {
		i++
		s := fmt.Sprintf("%016x", time.Now().UnixNano())
		kset := digest.SumUint64s([]byte(s))
		var d uint64
	probe:
		for d < dmax {
			for j := 0; j < h; j++ {
				k := kset[j]
				xof := (k + d) & mask
				if t[j][xof] == 0 {
					t[j][xof] = k
					continue next
				}
				d++
			}
			continue probe
		}
		break
	}
	loading := float64(i+1) / float64(size)
	log.Log("using %d H with tsize %d - %d probes at i:%d = %0.3f", h, size, dmax, i, loading)
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
func exitOnError(s string) {
	fmt.Fprintf(os.Stderr, s)
	os.Exit(1)
}
