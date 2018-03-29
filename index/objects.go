// Doost!

package index

import (
	"fmt"
	//	"io"
	//	"os"
	//	"path/filepath"
	//	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

// object.idx file is a sequential list of object ids. The adjusted offset of
// the fixed witdth Oid data (32B) is the implicit 'key' for the object. The
// adjustment is accounting for the objects.idx objectsHeader.
//
// On creation of new objects, an Oid entry is appended to the objects.idx file.
// The corresponding 'key' is recorded in the corresponding index.Card.
//
// On queries of objects for a given specification of tags (e.g AND or more
// selective logical expressions) an array of 'bits' is obtained from the Tagmap
// and these bit (positions) correspond to the 'keys' of object.idx, from which
// we maps the tagmap.bits -> object.keys -> Oids -> Cards.

func init() {
	// verify system size assumptions central to objects.idx file
	if system.OidSize != 32 {
		panic(errors.Fault("index/objects.go: Oid-Size:%d", system.OidSize))
	}

}

/// consts and vars ///////////////////////////////////////////////////////////

// object.idx file
const mmap_idx_file_code uint64 = 0x8fe452c6d1f55c66 // sha256("mmaped-index-file")[:8]
const idxFilename = "object.idx"                     // REVU belongs to toplevle gart package

// objectsHeader related consts
const (
	objectsHeaderSize = 0x1000
	objectsPageSize   = 0x1000
	objectsRecordSize = system.OidSize
)

/// object.idx file objectsHeader /////////////////////////////////////////////////////

type objectsHeader struct {
	ftype    uint64
	crc64    uint64 // objectsHeader crc
	created  int64
	updated  int64
	pcnt     uint64 // page count	=> pcnt
	ocnt     uint64 // record count => ocnt
	reserved [4048]byte
}

func (h *objectsHeader) Print() {
	fmt.Printf("file type:  %016x\n", h.ftype)
	fmt.Printf("crc64:      %016x\n", h.crc64)
	fmt.Printf("created:    %016x (%s)\n", h.created, time.Unix(0, h.created))
	fmt.Printf("updated:    %016x (%s)\n", h.updated, time.Unix(0, h.updated))
	fmt.Printf("page cnt: : %d\n", h.pcnt)
	fmt.Printf("object cnt: %d\n", h.ocnt)
}

func (h *objectsHeader) encode(buf []byte) error {
	if len(buf) < objectsHeaderSize {
		return errors.Error("objectsHeader.encode: insufficient buffer length: %d",
			len(buf))
	}

	*(*uint64)(unsafe.Pointer(&buf[0])) = h.ftype
	*(*int64)(unsafe.Pointer(&buf[16])) = h.created
	*(*int64)(unsafe.Pointer(&buf[24])) = h.updated
	*(*uint64)(unsafe.Pointer(&buf[32])) = h.pcnt
	*(*uint64)(unsafe.Pointer(&buf[40])) = h.ocnt

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64

	return nil
}

func (h *objectsHeader) decode(buf []byte) error {
	if len(buf) < objectsHeaderSize {
		return errors.Error("objectsHeader.decode: insufficient buffer length: %d",
			len(buf))
	}
	*h = *(*objectsHeader)(unsafe.Pointer(&buf[0]))

	/// verify //////////////////////////////////////////////////////

	if h.ftype != mmap_idx_file_code {
		return errors.Bug("objectsHeader.decode: invalid ftype: %x - expect: %x",
			h.ftype, mmap_idx_file_code)
	}
	crc64 := digest.Checksum64(buf[16:])
	if crc64 != h.crc64 {
		return errors.Bug("objectsHeader.decode: invalid checksum: %d - expect: %d",
			h.crc64, crc64)
	}
	if h.created == 0 {
		return errors.Bug("objectsHeader.decode: invalid created: %d", h.created)
	}
	if h.updated < h.created {
		return errors.Bug("objectsHeader.decode: invalid updated: %d < created:%d",
			h.updated, h.created)
	}

	return errors.NotImplemented("index.objectsHeader.decode")
}
