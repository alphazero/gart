package bitmap

import (
	"math/rand"
	"testing"
	"time"
)

var random *rand.Rand
var seed int64

func init() {
	seed = time.Now().UnixNano()
	//	seed = 1519249266045336101
	random = rand.New(rand.NewSource(seed))
}

const sparse_density = 0.07

func randomBitmap_7(size int) []byte {
	return randomBitmap_7_density(size, sparse_density)
}

// size is number of bytes.
// density [0.0, .9999..] is number of FORM bytes / size.
func randomBitmap_7_density(size int, density float64) []byte {
	var bitmap []byte
	p := int(1.0 / density)
	// generate bits in sequence
	for i := 0; i < size; i++ {
		var b byte
		if random.Intn(p) == 0 {
			b = byte(random.Intn(0x80))
		}
		bitmap = append(bitmap, b)
	}
	return bitmap
}

func TestBahCompressBAH7(t *testing.T) {
	bitmap := randomBitmap_7(64)
	bah := Compress(bitmap)
	DisplayBuf("bitmap", bitmap)
	DisplayBuf("bah07 ", bah)
	if bah == nil {
		t.Fatalf("Compress must never return nil")
	}
}

func TestBahDecompressBAH7(t *testing.T) {
	// test
	bitmap := randomBitmap_7(128 * 512)
	compressed := Compress(bitmap)
	result := Decompress(compressed)

	// result check
	type expectation struct {
		buf  []byte
		blen int
	}
	var expect = expectation{bitmap, len(bitmap)}
	var have = expectation{result, len(result)}

	if have.blen != expect.blen {
		t.Logf("len:%d", len(compressed))
		DisplayBuf("EXPECT", expect.buf)
		DisplayBuf("BAH", compressed)
		DisplayBuf("HAVE", have.buf)
		for i, b0 := range have.buf {
			if b := expect.buf[i]; b0 != b {
				t.Logf("  have[%d]:%08b expect[%d]:%08b <<", i, b0, i, b)
			} else {
				t.Logf("  have[%d]:%08b expect[%d]:%08b", i, b0, i, b)
			}
		}
		DisplayBuf("BAH ", compressed)
		t.Logf("SEED: %d", seed)
		t.Fatalf("have len: %d expect len:%d", have.blen, expect.blen)
	}

	for i, b := range expect.buf {
		if b0 := have.buf[i]; b0 != b {
			t.Fatalf("have byte[%d]:%08b expect byte[%d]:%08b", i, b0, i, b)
		}
	}
}

var benchmarks = []struct {
	name    string
	density float64
	mapsize int
}{
	/*
			{"100000000 0.000001", 0.000001, 100000000},
			{"100000000 0.00001", 0.00001, 100000000},
			{"100000000 0.0001", 0.0001, 100000000},
			{"100000000 0.001", 0.001, 100000000},
			{"100000000 0.01", 0.01, 100000000},

			{"20000000 0.000001", 0.000001, 20000000},
			{"20000000 0.00001", 0.00001, 20000000},
			{"20000000 0.0001", 0.0001, 20000000},
			{"20000000 0.001", 0.001, 20000000},
			{"20000000 0.01", 0.01, 20000000},
		{"4096 0.00001", 0.00001, 4096},
		{"4096 0.0001", 0.0001, 4096},
		{"4096 0.001", 0.001, 4096},
		{"4096 0.01", 0.01, 4096},
		{"4096 0.1", 0.1, 4096},

		{"2048 0.00001", 0.00001, 2048},
		{"2048 0.0001", 0.0001, 2048},
		{"2048 0.001", 0.001, 2048},
		{"2048 0.01", 0.01, 2048},
		{"2048 0.1", 0.1, 2048},

		{"1024 0.00001", 0.00001, 1024},
		{"1024 0.0001", 0.0001, 1024},
		{"1024 0.001", 0.001, 1024},
		{"1024 0.01", 0.01, 1024},
		{"1024 0.1", 0.1, 1024},

		{"512 0.00001", 0.00001, 512},
		{"512 0.0001", 0.0001, 512},
		{"512 0.001", 0.001, 512},
		{"512 0.01", 0.01, 512},
		{"512 0.1", 0.1, 512},

	*/
	{"256 0.00001", 0.00001, 256},
	{"256 0.0001", 0.0001, 256},
	{"256 0.001", 0.001, 256},
	{"256 0.01", 0.01, 256},
	{"256 0.1", 0.1, 256},

	{"128 0.00001", 0.00001, 128},
	{"128 0.0001", 0.0001, 128},
	{"128 0.001", 0.001, 128},
	{"128 0.01", 0.01, 128},
	{"128 0.1", 0.1, 128},

	{"64 0.00001", 0.00001, 64},
	{"64 0.0001", 0.0001, 64},
	{"64 0.001", 0.001, 64},
	{"64 0.01", 0.01, 64},
	{"64 0.1", 0.1, 64},

	{"32 0.00001", 0.00001, 32},
	{"32 0.0001", 0.0001, 32},
	{"32 0.001", 0.001, 32},
	{"32 0.01", 0.01, 32},
	{"32 0.1", 0.1, 32},

	{"16 0.00001", 0.00001, 16},
	{"16 0.0001", 0.0001, 16},
	{"16 0.001", 0.001, 16},
	{"16 0.01", 0.01, 16},
	{"16 0.1", 0.1, 16},

	{"8 0.00001", 0.00001, 8},
	{"8 0.0001", 0.0001, 8},
	{"8 0.001", 0.001, 8},
	{"8 0.01", 0.01, 8},
	{"8 0.1", 0.1, 8},

	{"4 0.00001", 0.00001, 4},
	{"4 0.0001", 0.0001, 4},
	{"4 0.001", 0.001, 4},
	{"4 0.01", 0.01, 4},
	{"4 0.1", 0.1, 4},
}

func BenchmarkCompress(b *testing.B) {
	for _, bm := range benchmarks {
		bitmap := randomBitmap_7_density(bm.mapsize, bm.density)
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Compress(bitmap)
			}
		})
	}
}

func BenchmarkDecompress(b *testing.B) {
	for _, bm := range benchmarks {
		bitmap := randomBitmap_7_density(bm.mapsize, bm.density)
		compressed := Compress(bitmap)
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Decompress(compressed)
			}
		})
	}
}

func randomSelection(n, from, to int) (arr []int) {
	p := (to - from) / (n)
	println(p)
	for i := from; i < to; i++ {
		if random.Intn(p) == 0 {
			arr = append(arr, i)
		}
	}
	//	log.Printf("%d\n", arr)
	return
}

/*
func TestGetBits(t *testing.T) {

	// 0  3    7           18     23     27 28   | 63 69
	// 0001000 1000000 0000100 1111111x7
	//         7       14      21     63
	bitmap := []byte{0x08, 0x40, 0x07, 0xc7}
	selection := []int{0, 3, 7, 13, 17, 18, 19, 20, 23, 27, 28, 70, 99}

	vals, oob := GetBits(bitmap, selection...)
	DisplayBuf("compressed", bitmap)
	DisplayBuf("decompress", Decompress(bitmap))

	t.Logf("\n%d\n%08b\n%t\n%d\n", selection, bitmap, vals, oob)
}

func BenchmarkGetBits(b *testing.B) {
	b.StopTimer()
	size := 32 << 2
	selection := randomSelection(5, 0, size*7)
	println(len(selection))
	//	println("--")
	bitmap := randomBitmap_7_density(size, 0.01)
	println(len(bitmap))
	compressed := Compress(bitmap)
	println(len(compressed))
	DisplayBuf("compress", compressed)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		GetBits(compressed, selection...)
	}
}
func BenchmarkSelectsAll(b *testing.B) {
	b.StopTimer()
	size := 32 << 2
	selection := randomSelection(9, 0, size*7)
	bitmap := randomBitmap_7_density(size, 0.01)
	compressed := Compress(bitmap)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		SelectsAll(compressed, selection...)
	}
}
*/
