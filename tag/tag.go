// Doost!

package tag

import (
	"fmt"
	"strings"
	"unsafe"
)

/// binary image invariants ///////////////////////////////////////////////////

// individual tag binary representation invariants
const (
	prefixBytes  = 6   // 1:flag 4:refcnt 1:value-len
	maxNameBytes = 255 // bytes not runes (e.g. non-latin unicode names)
	minTagBytes  = 8   //  prefix + 1 byte tag (e.g. '1')

	flagsOffset  = uintptr(0)
	refcntOffset = uintptr(1)
	lenOffset    = uintptr(5)
)

/// Tag ////////////////////////////////////////////////////////////////////////

type Tag struct {
	flags  byte   // reserved
	refcnt uint32 // number of objects associated with this tag
	name   string // see maxNameBytes

	/* not persisted in binary image -- ! maintain this order ! */
	id     int    // ids are ~7-encoded: for any id, id mod 8 != 0
	offset uint64 // offset + headerBytes = offset in file
}

// Returns length of the binary representation. For name len, just len(t.name).
func (t Tag) buflen() int { return prefixBytes + len(t.name) }

// semantic representation
func (t Tag) String() string { return fmt.Sprintf("tag:{ %q (%d) } ", t.name, t.refcnt) }

// debug representation
func (t Tag) Debug() string {
	return fmt.Sprintf("tag: %q refcnt:%d f:%08b id:%x binlen:%d",
		t.name, t.refcnt, t.flags, t.id, t.buflen())
}

func normalizeName(name string) (string, bool) {
	name = strings.ToLower(name) // in case this affects len for funky langs
	length := len(name)
	if length == 0 || length > maxNameBytes {
		return "", false
	}
	return name, true
}

// Tag name must be at most maxNameBytes long. Zerolen strings are not permitted.
// Tag name is stored in lower-case, regardless of the input arg case.
func newTag(tag string, id int, offset uint64) (*Tag, error) {
	name, ok := normalizeName(tag)
	if !ok {
		return nil, fmt.Errorf("Tag.newTag: invalid argument - name %q", tag)
	}
	return &Tag{name: name, id: id, offset: offset}, nil
}

// decode reads a Tag from the provided buffer, returning the number of bytes
// read.  The provided slice must be at least minTagBytes bytes in length.
//
// Function returns error if b is nil or does not meet the minimum length requirement.
func (t *Tag) decode(b []byte) (int, error) {
	println("decode")
	if b == nil {
		return 0, fmt.Errorf("Tag.decode: invalid argument - b is nil")
	}
	if len(b) < minTagBytes {
		return 0, fmt.Errorf("Tag.decode: invalid argument - len(b) < %d ", minTagBytes)
	}

	t.flags = b[flagsOffset]
	t.refcnt = *(*uint32)(unsafe.Pointer(&b[refcntOffset]))
	namelen := int(b[lenOffset])
	n := prefixBytes + int(namelen)
	t.name = string(b[prefixBytes:n])

	return n, nil
}

// Encode writes the binary representation of a Tag to the provided buffer. The
// number of bytes written is returned if no error encountered. buflength of the in arg
// slice 'b' must be >= tag.buflen()
//
// Nil and undersized buffers will result in errors.
func (t Tag) encode(b []byte) (int, error) {
	println("encode")
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
	if n := copy(b[prefixBytes:], []byte(t.name)); n != namelen {
		panic(fmt.Sprintf("bug - only copied %d bytes of name (len:%d)", n, t.buflen()))
	}

	// XXX
	fmt.Printf("debug: Tag.encode: f:%d refcnt:%08x len:%d name:%q\n                  ", t.flags, t.refcnt, namelen, t.name)
	for i := 0; i < t.buflen(); i++ {
		fmt.Printf(" %02x", b[i])
	}
	fmt.Println()
	// XXX

	return t.buflen(), nil
}
