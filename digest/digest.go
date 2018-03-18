// Friend

package digest

import (
	"blake2b"
	"hash/crc32"
	"hash/crc64"
	"unsafe"

	"github.com/alphazero/gart/fs"
)

/// consts and vars ///////////////////////////////////////////////////////////

const (
	HashBytes = 32
)

/// consts and vars ///////////////////////////////////////////////////////////

// Return first 8 bytes of a Black2B Sum as a uint64 value
// NOTE at around ~460ns/op this is relatively slow, but intended use is
//      creating certain to be unique 64bit keys for tag-names, which are
//      created once per tag in the life-time of gart.
//
// NOTE the endian-ness of using unsafe flips the bytes so
//      h:   ffeeddccbbaa9988........ the full b2b hash
//      n: 0x8899aabbccddeeff
func SumUint64(b []byte) uint64 {
	h := blake2b.Sum256(b)
	return *(*uint64)(unsafe.Pointer(&h[0]))
}

// Sum: Black2B size 256 digest
func Sum(b []byte) [HashBytes]byte {
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
