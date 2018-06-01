// Doost !

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/alphazero/gart/syslib/bitmap"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

/// adhoc test /////////////////////////////////////////////////////////////////

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	debug.Printf("debug is working - %d", system.MaxTagNameSize)
	fixXorBug()

	// to try:
	// - find optimal way to use []int32 for wah
	// - sketch out Wahl 32-bit encoding, compression, and logical ops
	// Thank you FRIEND! Done!
	//	tryUncompressed()
	tryCompressed()
	/* REVU done - Thanks!
	 */
	tryQuery()
}

func exitOnError(e error) {
	fmt.Printf("err - %v\n", e)
	os.Exit(1)
}

func fixXorBug() {
	var win = getWahl(29, 0x8000)
	win.Compress()
	win.Print(os.Stdout)
	var wex = getWahl(0, 29)
	wex.Compress()
	wex.Print(os.Stdout)

	xor, e := win.Or(wex)
	if e != nil {
		exitOnError(e)
	}
	xor.Compress()
	xor.Print(os.Stdout)
}

func getWahl(n0, cnt uint) *bitmap.Wahl {
	var w = bitmap.NewWahl()
	var bits = make([]uint, cnt)
	for i := uint(0); i < cnt; i++ {
		bits[i] = n0 + i
	}
	w.Set(bits...)
	verifySet(w, bits)
	return w
}

func getRandomWahl(max uint) *bitmap.Wahl {
	var w = bitmap.NewWahl()
	var bits []uint
	for i := uint(0); i < max; i++ {
		if rnd.Intn(100) > 50 {
			bits = append(bits, i)
		}
	}
	w.Set(bits...)
	return w
}

func tryCompressed() {
	// lotsofones are bits in range (1000, 1332)
	lotsofones := make([]uint, 333)
	for i := 0; i < len(lotsofones); i++ {
		lotsofones[i] = uint(i + 1000)
	}

	var bits []uint

	var wahl_1 = bitmap.NewWahl()
	bits = []uint{0, 30, 63, 93, 3333}
	wahl_1.Set(bits...)
	verifySet(wahl_1, bits)

	bits = lotsofones[:333]
	wahl_1.Set(bits...)
	verifySet(wahl_1, bits)

	wahl_1.Bits().Print(os.Stdout)

	bits = []uint{0, 30, 63, 93}
	bits = append(bits, lotsofones[:333]...)
	wahl_1.Compress()
	verifySet(wahl_1, bits)

	//	wahl_1.Print(os.Stdout)

	var wahl_2 = bitmap.NewWahl()
	wahl_2.Set(0, 1, 29, 31, 93, 124, 155, 185, 186, 1000, 1001, 1003, 1007, 2309, 2311)
	wahl_2.Set(lotsofones[:111]...)
	wahl_2.Bits().Print(os.Stdout)
	wahl_2.Compress()
	//	wahl_2.Print(os.Stdout)

	// test NOT
	fmt.Printf("\n=== TEST NOT ====================\n")
	wahl_1_not := wahl_1.Not()
	wahl_1_not.Bits().Print(os.Stdout)
	//	wahl_1_not.Print(os.Stdout)
	verifyNot(wahl_1, wahl_1_not)

	// test AND
	fmt.Printf("\n=== TEST AND ====================\n")
	wahl_1_and_2, e := wahl_1.And(wahl_2)
	if e != nil {
		exitOnError(e)
	}
	fmt.Print("\nWAHL-1: \n")
	//	wahl_1.Bits().Print(os.Stdout)
	wahl_1.Print(os.Stdout)
	fmt.Print("\nWAHL-2: \n")
	//	wahl_2.Bits().Print(os.Stdout)
	wahl_2.Print(os.Stdout)
	fmt.Print("\nAND: \n")
	//	wahl_1_and_2.Bits().Print(os.Stdout)
	wahl_1_and_2.Print(os.Stdout)
	verifyAnd(wahl_1, wahl_2, wahl_1_and_2)

	// test OR
	fmt.Printf("\n=== TEST OR =====================\n")
	wahl_1_or_2, e := wahl_1.Or(wahl_2)
	if e != nil {
		exitOnError(e)
	}
	fmt.Print("\nWAHL-1: \n")
	//	wahl_1.Bits().Print(os.Stdout)
	wahl_1.Print(os.Stdout)
	fmt.Print("\nWAHL-2: \n")
	//	wahl_2.Bits().Print(os.Stdout)
	wahl_2.Print(os.Stdout)
	fmt.Print("\nOR : \n")
	//	wahl_1_or_2.Bits().Print(os.Stdout)
	wahl_1_or_2.Print(os.Stdout)
	verifyOr(wahl_1, wahl_2, wahl_1_or_2)

	// test Clear
	fmt.Printf("\n=== TEST Clear ==================\n")
	fmt.Printf("=== clear anded bitmap ==========\n")
	bits = []uint{0, 95, 222, 1023, 1025, 1027}
	wahl_1_and_2.Clear(bits...)
	verifyClear(wahl_1_and_2, bits)

	wahl_1_and_2.Compress()
	verifyClear(wahl_1_and_2, bits)

	wahl_1_and_2.Bits().Print(os.Stdout)
	wahl_1_and_2.Print(os.Stdout)

	fmt.Printf("\n=== clear or'd bitmap -==========\n")
	bits = []uint{1, 30, 1023, 1052, 1111, 1300, 1302}
	wahl_1_or_2.Clear(bits...)
	verifyClear(wahl_1_or_2, bits)

	wahl_1_or_2.Compress()
	verifyClear(wahl_1_or_2, bits)

	wahl_1_or_2.Bits().Print(os.Stdout)
	wahl_1_or_2.Print(os.Stdout)

	return
}

// check query plan
// i - OR all the excluded
// ii - NOT the result
// iii - AND with all
// iv - AND with included
func tryQuery() {
	fmt.Printf("=== query ======================\n")
	var n = uint(rnd.Intn(1 << 8))

	fmt.Printf("--- all ---\n")
	var all = getWahl(0, n)
	all.Compress()
	all.Print(os.Stdout)

	var xn = rnd.Intn(10)
	var xlist = make([]*bitmap.Wahl, xn)
	for i := 0; i < xn; i++ {
		fmt.Printf("--- exclude ---\n")
		xlist[i] = getRandomWahl(n)
		xlist[i].Compress()
		xlist[i].Print(os.Stdout)
	}

	var in = rnd.Intn(10)
	var ilist = make([]*bitmap.Wahl, in)
	for i := 0; i < in; i++ {
		fmt.Printf("--- include ---\n")
		ilist[i] = getRandomWahl(n)
		ilist[i].Compress()
		ilist[i].Print(os.Stdout)
	}

	x0, e := bitmap.Or(xlist...)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("--- x0 exclude ---\n")
	x0.Print(os.Stdout)

	x0 = x0.Not()
	fmt.Printf("--- x0 exclude not ---\n")
	x0.Print(os.Stdout)

	xf, e := x0.And(all)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("--- xf filter ----\n")
	all.Print(os.Stdout)
	xf.Print(os.Stdout)
	verifyAnd(x0, all, xf)

	i0, e := bitmap.And(ilist...)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("--- i0 include ---\n")
	i0.Print(os.Stdout)

	res, e := i0.And(xf)
	if e != nil {
		exitOnError(e)
	}

	// TODO verify
	fmt.Printf("=== query result ===============\n")
	res.Print(os.Stdout)
}

func tryUncompressed() {
	// lotsofones are bits in range (1000, 1332)
	lotsofones := make([]uint, 333)
	for i := 0; i < len(lotsofones); i++ {
		lotsofones[i] = uint(i + 1000)
	}

	var wahl_1 = bitmap.NewWahl()
	wahl_1.Set(0, 30, 63, 93)
	wahl_1.Set(lotsofones[:33]...)
	wahl_1.Bits().Print(os.Stdout)
	wahl_1.Print(os.Stdout)

	var wahl_2 = bitmap.NewWahl()
	wahl_2.Set(0, 1, 29, 31, 93, 124, 155, 185, 186, 1000, 1001, 1003, 1007, 2309, 2311)
	wahl_2.Bits().Print(os.Stdout)
	wahl_2.Print(os.Stdout)

	fmt.Printf("\n=== TEST AND =( uncompressed )===\n")
	wahl_1_and_2, e := wahl_1.And(wahl_2)
	if e != nil {
		exitOnError(e)
	}
	wahl_1_and_2.Bits().Print(os.Stdout)
	wahl_1_and_2.Print(os.Stdout)

	return
}

/// helpers //////////////////////////////////////////////////////

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

func verifyNot(w, wnot *bitmap.Wahl) {
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
			panic(errors.Bug("NOT: bit %d is in both\n", n))
		}
	}
}

// and.Max must be equal to min(a.Max, b.Max)
func verifyAnd(a, b, and *bitmap.Wahl) {
	a_map := mapArray(a.Bits())
	b_map := mapArray(b.Bits())
	and_map := mapArray(and.Bits())
	ref_map := andMaps(a_map, b_map)
	compareMaps("verify AND map", and_map, ref_map)
	for _, bit := range and.Bits() {
		// bit must be in both maps for AND
		if !(a_map[bit] && b_map[bit]) {
			panic(errors.Bug("AND: bit %d is not in both maps\n", bit))
		}
	}
}

// asserts maps are identical: have same length and same content
func compareMaps(info string, a, b map[int]bool) {
	if len(a) != len(b) {
		panic(errors.Bug("%s - %d != %d", info, len(a), len(b)))
	}
	for k, v := range a {
		if _, ok := b[k]; !ok {
			panic(fmt.Sprintf("%s - k:%d (%t)\n", info, k, v))
		}
	}
}

// or.Max must be equal to max(a.Max, b.Max)
func verifyOr(a, b, or *bitmap.Wahl) {
	a_map := mapArray(a.Bits())
	b_map := mapArray(b.Bits())
	or_map := mapArray(or.Bits())
	ref_map := orMaps(a_map, b_map)
	compareMaps("verify OR map", or_map, ref_map)
	ab_max := max(a.Max(), b.Max())
	if or.Max() != ab_max {
		panic(errors.Bug("OR: tail-error - Max(a:%d, b:%d) and or.Max:%d\n",
			a.Max(), b.Max(), or.Max()))
	}
	for _, bit := range or.Bits() {
		// bit must be in both maps for AND
		if !(a_map[bit] || b_map[bit]) {
			panic(errors.Bug("OR: bit %d is not in either maps\n", bit))
		}
	}
}

func smallerFirst(a, b []int) ([]int, []int) {
	if len(a) < len(b) {
		return a, b
	}
	return b, a
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
