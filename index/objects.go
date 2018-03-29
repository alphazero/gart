// Doost!

package index

import (
	"fmt"
	//	"io"
	"os"
	"path/filepath"
	//	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
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

/// consts and vars ///////////////////////////////////////////////////////////

// object.idx file
const (
	mmap_idx_file_code uint64 = 0x8fe452c6d1f55c66 // sha256("mmaped-index-file")[:8]
	oidxFileBasename          = "object.idx"       // REVU belongs to toplevle gart package
)

var oidxFilename string // set by init()

// objectsHeader related consts
const (
	objectsHeaderSize = 0x1000
	objectsPageSize   = 0x1000
	objectsRecordSize = system.OidSize
)

/// object.idx specific inits //////////////////////////////////////////////////

func init() {
	// verify system size assumptions central to objects.idx file
	if system.OidSize != 32 {
		panic(errors.Fault("index/objects.go: Oid-Size:%d", system.OidSize))
	}

	oidxFilename = filepath.Join(system.IndexObjectsPath, oidxFileBasename)
}

/// object.idx file objectsHeader /////////////////////////////////////////////////////

// object.idx file's header is a page size (4KB) structure and is the minimal object
// index file. The crc64 field is the checksum of the header only. The reserved bits
// are for a projected merkel checksum of actual object index data chunks.
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

/// object.idx file ////////////////////////////////////////////////////////////

// oidxFile structure captures the persistent and run-time meta-data and data of
// the object.idx file. The file is memory mapped and supports distinct opModes.
// This structure and associated functions and the logical object index itself
// are only used by the index package and not top-level gart tools (yet).
type oidxFile struct {
	*objectsHeader
	filename string
	opMode   OpMode
	file     *os.File
	finfo    os.FileInfo // size is int64
	flags    int
	prot     int
	buf      []byte
	offset   int64
	modified bool
}

// CreateObjectIndex creates the initial (header only/empty) object.idx file.
// The file is closed on return.
//
// Returns index.ErrObjectIndexExists if index file already exists.
func createObjectIndex() error {

	// for convenience
	errorWithCause := errors.ErrorWithCause

	// create object index file
	file, e := fs.OpenNewFile(oidxFilename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return errorWithCause(e, "oidx.CreateIndex")
	}
	defer file.Close()

	// encode and write header
	var now = time.Now().UnixNano()
	var header = &objectsHeader{
		ftype:   mmap_idx_file_code, // TODO uniformly call these _file
		created: now,
		updated: now,
		pcnt:    0,
		ocnt:    0,
	}
	var buf [objectsHeaderSize]byte
	if e := header.encode(buf[:]); e != nil {
		if ec := os.Remove(oidxFilename); ec != nil {
			panic(errors.Fault(
				"oidx.CreateIndex: os.Remove - %s - while recovering from: %s", ec, e))
		}
		return errorWithCause(e, "oidx.CreateIndex")
	}

	_, e = file.Write(buf[:])
	if e != nil {
		return errorWithCause(e, "oidx.CreateIndex")
	}

	return nil
}

// OpenObjectIndex opens the objects.idx in the given OpMode and returns
// the handle to the index.
//
// Returns index.ErrObjectIndexNotExist if the index does not exist.
// Function also returns any other error encountered in its execution.
// In case of error results, the oidxFile pointer will be nil and file closed.
func openObjectIndex(opMode OpMode) (*oidxFile, error) {
	return nil, errors.NotImplemented("index.OpenObjectIndex")
}

// closeIndex closes the index, at which point the reference to the pointer
// should be discarded.
//
// Returns index.ErrObjectIndexClosed if index has already been closed.
func (oidx *oidxFile) closeIndex() error {
	return errors.NotImplemented("oidxFile.Close")
}

// addObject appends the given oid to the object index. Index must have been
// openned in OpMode#Write. The underlying file will be extended by a page
// if required.
//
//
func (oidx *oidxFile) addObject(oid []byte) error {
	return errors.NotImplemented("oidxFile.AddObject")
}

// lookupOidByKey returns a mapping of uint64 keys to []byte slice data of
// object ids (Oid.data). The returning mapping may have less elements than
// the number of input keys if the index does not contain a mapping for the
// key.
//
// Returns (map, index.ErrNoSuchObject) if one or more of the input keys are not bound.
// The map may still contain other mappings and will not be nil though possibly
// empty.
//
// Returns index.ErrObjectIndexClosed or any other encountered error, in which
// the resultant map will be nil.
func (oidx *oidxFile) lookupOidByKey(key ...uint64) (map[uint64][]byte, error) {
	return nil, errors.NotImplemented("oidxFile.Lookup")
}
