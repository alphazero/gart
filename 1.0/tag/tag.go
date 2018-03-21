// Doost!

package tag

import (
	"fmt"
	"strings"
	"unsafe"
)

/// Tag ////////////////////////////////////////////////////////////////////////

// individual tag binary representation invariants
const (
	prefixSize  = 6   // 1:flag 4:refcnt 1:value-len
	maxNameSize = 255 // bytes not runes (e.g. non-latin unicode names)
	minTagSize  = 8   //  prefix + 1 byte tag (e.g. '1')

	flagsOffset  = uintptr(0)
	refcntOffset = uintptr(1)
	lenOffset    = uintptr(5)
)

type Tag struct {
	flags  byte   // reserved
	refcnt uint32 // number of objects associated with this tag
	name   string // see maxNameSize

	/* not persisted in binary image -- ! maintain this order ! */
	id     int    // ids are ~7-encoded: for any id, id mod 8 != 0
	offset uint64 // offset + headerSize = offset in file
}

// Returns length of the binary representation. For name len, just len(t.name).
func (t Tag) buflen() int { return prefixSize + len(t.name) }

// semantic representation
func (t Tag) String() string { return fmt.Sprintf("tag:{ %q (%d) } ", t.name, t.refcnt) }

// debug representation
func (t Tag) Debug() string {
	return fmt.Sprintf("tag: f:%08b id:%4d refcnt:%4d binlen:%d %q",
		t.flags, t.id, t.refcnt, t.buflen(), t.name)
}

func normalizeName(name string) (string, bool) {
	name = strings.ToLower(name) // in case this affects len for funky langs
	length := len(name)
	if length == 0 || length > maxNameSize {
		return "", false
	}
	return name, true
}

// Tag name must be at most maxNameSize long. Zerolen strings are not permitted.
// Tag name is stored in lower-case, regardless of the input arg case.
func newTag(tag string, id int, offset uint64) (*Tag, error) {
	name, ok := normalizeName(tag)
	if !ok {
		return nil, fmt.Errorf("Tag.newTag: invalid argument - name %q", tag)
	}
	return &Tag{name: name, id: id, offset: offset}, nil
}

// decode reads a Tag from the provided buffer, returning the number of bytes
// read.  The provided slice must be at least minTagSize bytes in length.
//
// Function returns error if b is nil or does not meet the minimum length requirement.
func (t *Tag) decode(b []byte) (int, error) {
	if b == nil {
		return 0, fmt.Errorf("Tag.decode: invalid argument - b is nil")
	}
	if len(b) < minTagSize {
		return 0, fmt.Errorf("Tag.decode: invalid argument - len(b) < %d ", minTagSize)
	}

	t.flags = b[flagsOffset]
	t.refcnt = *(*uint32)(unsafe.Pointer(&b[refcntOffset]))
	namelen := int(b[lenOffset])
	n := prefixSize + int(namelen)
	t.name = string(b[prefixSize:n])

	return n, nil
}

// Encode writes the binary representation of a Tag to the provided buffer. The
// number of bytes written is returned if no error encountered. buflength of the in arg
// slice 'b' must be >= tag.buflen()
//
// Nil and undersized buffers will result in errors.
func (t Tag) encode(b []byte) (int, error) {
	if b == nil {
		return 0, fmt.Errorf("Tag.Encode: invalid argument - b is nil")
	}
	if len(b) < t.buflen() {
		return 0, fmt.Errorf("Tag.Encode: invalid argument - len(b) < %d ", t.buflen())
	}

	b[flagsOffset] = t.flags
	var namelen = len(t.name)
	*(*uint32)(unsafe.Pointer(&b[refcntOffset])) = t.refcnt
	b[lenOffset] = byte(namelen)
	if n := copy(b[prefixSize:], []byte(t.name)); n != namelen {
		panic(fmt.Sprintf("bug - only copied %d bytes of name (len:%d)", n, t.buflen()))
	}

	return t.buflen(), nil
}
