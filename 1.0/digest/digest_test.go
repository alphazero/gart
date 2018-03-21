package digest_test

import (
	"testing"

	"github.com/alphazero/gart/digest"
)

const smallfile = "digest_test.go"
const largefile = "testfile.big"

var path = []byte("/Users/alphazero/Code/oss/halide-tutorial-code-CVPR2015/.git/objects/pack/pack-32b76872a71454dfc48ce7ffa328fdefd8379e46.pack")

func BenchmarkBlake2bPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		md := digest.Sum(path)
		if len(md) != 32 {
			b.Fatalf("err - len: expected:32 have:%d", len(md))
		}
	}
}
func BenchmarkBlake2bLargeFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, e := digest.SumFile(largefile)
		if e != nil {
			b.Fatalf("err = %v", e)
		}
	}
}
func BenchmarkBlake2bSmallFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, e := digest.SumFile(smallfile)
		if e != nil {
			b.Fatalf("err = %v", e)
		}
	}
}
func BenchmarkBlake2bSumUint64(b *testing.B) {
	var tagname = []byte("this is a relatively long tag name")
	for i := 0; i < b.N; i++ {
		digest.SumUint64(tagname)
	}
}
