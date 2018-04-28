// Doost!

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/system/log"
)

var b int = 1
var d uint = 12
var h int = 4
var pmin int = 256
var pmax int = 256

func init() {
	flag.IntVar(&b, "b", b, "stride - min 1")
	flag.UintVar(&d, "d", d, "d (degree) determines total capacity at 2^d or (1 << d)")
	flag.IntVar(&h, "h", h, "number of hash functions - min 1")
	flag.IntVar(&pmin, "p", pmin, "minimum number of probes")
	flag.IntVar(&pmax, "P", pmax, "maximum number of probes")
}

func main() {
	flag.Parse()
	log.Verbose(os.Stdout)
	log.Log("Salaam Samad Sultan of LOVE!")

	if h < 1 || h > 8 {
		exitOnError("h must be in range (1, 8) inclusive.")
	}

	if b < 1 {
		exitOnError("b must be > 0")
	}

	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	var lsum float64
	var tsum int64
	var i int
test:
	for ; i < 100; i++ {
		select {
		case <-c:
			break test
		default:
			loading, ait := linearProbe(i)
			lsum += loading
			tsum += ait
		}
	}
	avg := lsum / float64(i)
	ait := tsum / int64(i)
	log.Log("\naverage (%d runs) loading: %0.4f avg insert time: %d", i, avg, ait)
}

func cuckoo(n int) (float64, int64) {
	// cuckoo hash
	// the on-disk hash should be cuckoo since that has guaranteed 2 page loads per lookup
	// if we make it a 128 slot bucket, it is basically like linear probing 128 cells. the
	// only issue is that comparing 32 byte arrays can be expensive so it is 4 uint64 compares
	// per cell.
	//
	// this is what should be tested for timing. if it is OK then for all Read ops, we have a
	// solution for the index. But updating this fs hash on Write ops is probably not optimal.
	// For Write ops, the index should be converted into an in-memory linear-probe hashmap.
	// On save, we store as cuckoo.
	//
	// this is crazy. Isn't B+-Tree the solution for this specific problem, Joubin?
	//
	return 0.0, 0
}

func linearProbe(n int) (float64, int64) {
	var size = 1 << d
	var t [][]uint64 = make([][]uint64, h)
	var size0 = size / h //len(t)
	var mask = uint64(size0 - 1)
	var dmax = max(uint64(size0/100), uint64(pmin))
	dmax = min(dmax, uint64(pmax))
	dmax = min(dmax, uint64(size))
	if n == 0 {
		log.Log("using %d H & %d sized segments for total capacity of %d with stride %d & max probe %d", h, size0, size, b, dmax)
	}
	for x := 0; x < h; x++ {
		t[x] = make([]uint64, size0)
	}

	var i int
	var tsum int64
next:
	for i < size {
		i++
		s := fmt.Sprintf("%016x", time.Now().UnixNano())
		kset := digest.SumUint64s([]byte(s))
		var t0 = time.Now().UnixNano()
		var d uint64
	probe:
		for d < dmax {
			for j := 0; j < h; j++ {
				// stride cache lines for faster execution
				for l := 0; l < b; l++ {
					k := kset[j]
					xof := (k + d) & mask
					if t[j][xof] == 0 {
						t[j][xof] = k
						tsum += (time.Now().UnixNano() - t0)
						continue next
					}
					d++
				}
			}
			continue probe
		}
		break
	}
	loading := float64(i+1) / float64(size)
	print(".")
	//	log.Log("stop at %d probes at i: %d loading: %0.3f", dmax, i, loading)
	return loading, tsum / int64(i+1)
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
func exitOnError(s string) {
	fmt.Fprintf(os.Stderr, s+"\n")
	os.Exit(1)
}
