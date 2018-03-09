// Friend

package digest

import (
	"blake2b"
	"hash/crc32"
	"hash/crc64"

	"github.com/alphazero/gart/fs"
)

// Sum: Black2B size 256 digest
func Sum(b []byte) [32]byte {
	return blake2b.Sum256(b)
}

// Returns the (32 byte) Blake2B digest of the named file.
func SumFile(fname string) ([]byte, error) {
	buf, e := fs.ReadFull(fname)
	if e != nil {
		return nil, e
	}

	h := Sum(buf)
	return h[:], nil
}

/// checksums /////////////////////////////////////////////////////////////////

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

/// blob digests //////////////////////////////////////////////////////////////
