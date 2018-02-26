package digest_test

import (
	"archive/digest"
	"crypto"
	"log"
	"testing"
)

const fname = "digest_test.go"

func TestSha1(t *testing.T) {
	md, e := digest.ComputeWith(fname, crypto.SHA1)
	if e != nil {
		t.Fatalf("error -> %v", e)
	}
	if md == nil {
		t.Fatalf("md is nil")
	}
	log.Printf("md: len:%d %x %x %x\n", len(md), md, md[:1], md[1:])
}

func BenchmarkSha1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, e := digest.ComputeWith(fname, crypto.SHA1)
		if e != nil {
			b.Fatalf("err - %v", e)
		}
	}
}

func BenchmarkSha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, e := digest.ComputeWith(fname, crypto.SHA256)
		if e != nil {
			b.Fatalf("err = %v", e)
		}
	}
}

func BenchmarkSha512(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, e := digest.ComputeWith(fname, crypto.SHA512)
		if e != nil {
			b.Fatalf("err = %v", e)
		}
	}
}
