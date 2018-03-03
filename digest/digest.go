// Friend

package digest

import (
	"crypto"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
	"os"
)

/// system wide ///////////////////////////////////////////////////////////////

// crc table Qs
const (
	crc64qPoly = 0xdb5a6289da3a511e
	crc32qPoly = 0xdb5a62B9
)

// lazy init on demand
var (
	crc64q *crc64.Table
	crc32q *crc32.Table
)

// panics on nil arg.
func Checksum64(b []byte) uint64 {
	if b == nil {
		panic("bug - digest.Checksum64 - b is nil")
	}
	if crc64q == nil {
		crc64q = crc64.MakeTable(crc64qPoly)
	}
	return crc64.Checksum(b, crc64q)
}

// panics on nil arg.
func Checksum32(b []byte) uint32 {
	if b == nil {
		panic("bug - digest.Checksum32 - b is nil")
	}
	if crc32q == nil {
		crc32q = crc32.MakeTable(crc32qPoly)
	}
	return crc32.Checksum(b, crc32q)
}

/// types /////////////////////////////////////////////////////////////////////
// insure these cryptos are linked via import. Such a goofy language.
var (
	_ = sha1.New
	_ = sha256.New
	_ = sha512.New
)

// Default uses sha-1
func Compute(fname string) ([]byte, error) {
	return ComputeWith(fname, crypto.SHA1)
}

func ComputeWith(fname string, hash crypto.Hash) (md []byte, err error) {
	defer func(e *error) {
		r := recover()
		if r != nil {
			*e = fmt.Errorf("(recovered) %v", r)
		}
	}(&err)

	// note: hash.New() can panic if crypto package not linked.
	return compute(fname, hash.New())
}

func compute(fname string, h hash.Hash) ([]byte, error) {
	f, e := os.Open(fname)
	if e != nil {
		return nil, e
	}
	defer f.Close()

	if _, e := io.Copy(h, f); e != nil {
		return nil, e
	}
	return h.Sum(nil), nil
}
