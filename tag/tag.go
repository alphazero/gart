// Doost!

package tag

import (
	"fmt"
	"unsafe"
)

const (
	// flag, refcnt & 1 byte name len. Id is not persisted.
	prefixLen = 6

	// maximum tag name bytes length
	maxNameBytesLen = 256 - prefixLen

	// The minimal binary representation of a Tag with a single byte name
	minBinaryRepLen = 8
)

// Tag structure defines the in-memory representation of a gart tag.
type Tag struct {
	id     int
	flags  byte   // reserved
	refcnt uint32 // number of files tagged with this tag
	name   string // constraint: max len is 250 bytes.
}

/*
func (t Tag) Name() string  { return t.name }
func (t Tag) Flags() byte    { return t.flags }
func (t Tag) Id() int        { return t.id }
func (t Tag) RefCnt() uint32 { return t.refcnt }
*/
func (t Tag) Len() int       { return len([]byte(t.name)) }
func (t Tag) binaryLen() int { return prefixLen + len([]byte(t.name)) }

func (t Tag) String() string {
	return fmt.Sprintf("flags:%08b id:%x refcnt:%d vlen:%d name:%q",
		t.flags, t.id, t.refcnt, t.Len(), t.name)
}

// decode reads a Tag from the provided buffer, returning the number of bytes
// read.  The provided slice must be at least minBinaryRepLen bytes in length.
//
// Function returns error if b is nil or does not meet the minimum length requirement.
func (t *Tag) decode(b []byte) (int, error) {
	if b == nil {
		return 0, fmt.Errorf("Tag.Decode: invalid argument - b is nil")
	}
	if len(b) < minBinaryRepLen {
		return 0, fmt.Errorf("Tag.Decode: invalid argument - len(b) < %d ", minBinaryRepLen)
	}

	t.flags = b[0]
	t.refcnt = *(*uint32)(unsafe.Pointer(&b[1]))
	vlen := b[5]
	n := 6 + int(vlen)
	t.name = string(b[6:n])

	return n, nil
}

// Encode writes the binary representation of a Tag to the provided buffer. The
// number of bytes written is returned if no error encountered. Length of the in arg
// slice 'b' must be >= tag.Len()
//
// Nil and undersized buffers will result in errors.
func (t Tag) encode(b []byte) (int, error) {
	if b == nil {
		return 0, fmt.Errorf("Tag.Encode: invalid argument - b is nil")
	}
	if len(b) < t.binaryLen() {
		return 0, fmt.Errorf("Tag.Encode: invalid argument - len(b) < %d ", t.binaryLen())
	}
	b[0] = t.flags
	*(*uint32)(unsafe.Pointer(&b[1])) = t.refcnt
	b[5] = byte(t.Len())
	if n := copy(b[6:], []byte(t.name)); n != t.Len() {
		panic(fmt.Sprintf("bug - only copied %d bytes of name (len:%d)", n, t.Len()))
	}
	return t.binaryLen(), nil
}
