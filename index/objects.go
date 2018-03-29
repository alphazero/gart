// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
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

func (h *objectsHeader) Print(w io.Writer) {
	fmt.Fprintf(w, "file type:  %016x\n", h.ftype)
	fmt.Fprintf(w, "crc64:      %016x\n", h.crc64)
	fmt.Fprintf(w, "created:    %016x (%s)\n", h.created, time.Unix(0, h.created))
	fmt.Fprintf(w, "updated:    %016x (%s)\n", h.updated, time.Unix(0, h.updated))
	fmt.Fprintf(w, "page cnt: : %d\n", h.pcnt)
	fmt.Fprintf(w, "object cnt: %d\n", h.ocnt)
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
	header   *objectsHeader
	opMode   OpMode
	source   string
	file     *os.File
	finfo    os.FileInfo // size is int64
	flags    int
	prot     int
	buf      []byte
	offset   int64
	modified bool
}

func (oidx *oidxFile) Print(w io.Writer) {
}

func (oidx *oidxFile) hexdump(w io.Writer, page uint64) {
	var p = int(page)
	var width = 16
	var poff = p << 12
	var pend = poff + 0x1000 // page size 4K
	for i := poff; i < pend; i += width {
		fmt.Fprintf(w, "%08x % 02x\n", i, oidx.buf[i:i+width])
	}
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

	/// verify opmod and init accordingly ///////////////////////////

	if e := opMode.verify(); e != nil {
		return nil, e
	}

	var flags int
	var oflags int
	var prot int
	switch opMode {
	case Read:
		flags = syscall.MAP_PRIVATE
		prot = syscall.PROT_READ
		oflags = os.O_RDONLY
	case Write:
		flags = syscall.MAP_SHARED
		prot = syscall.PROT_WRITE
		oflags = os.O_RDWR
	default:
		panic("bug - unsupported opMode")
	}

	/// open file and map objectFile /////////////////////////////////

	file, e := os.OpenFile(oidxFilename, oflags, system.FilePerm)
	if e != nil {
		return nil, e
	}
	// note: close file on any error below
	finfo, e := file.Stat()
	if e != nil {
		file.Close()
		return nil, e
	}

	var oidx = &oidxFile{
		source: oidxFilename,
		opMode: opMode,
		file:   file,
		finfo:  finfo,
		prot:   prot,
		flags:  flags,
	}

	var offset int64 = 0
	if e := oidx.mmap(offset, int(finfo.Size()), false); e != nil {
		file.Close()
		return nil, e
	}

	return oidx, nil
}

// closeIndex closes the index, at which point the reference to the pointer
// should be discarded.
//
// Returns index.ErrObjectIndexClosed if index has already been closed.
func (oidx *oidxFile) closeIndex() error {
	return oidx.unmapAndClose()
}

// addObject appends the given oid to the object index. Index must have been
// openned in OpMode#Write. The underlying file will be extended by a page
// if required.
//
//
func (oidx *oidxFile) addObject(oid []byte) error {
	if oidx.opMode != Write {
		return errors.Bug("oidxFile.AddObject: invalid op-mode:%s", oidx.opMode)
	}

	var offset = ((oidx.header.ocnt) << 5) + objectsHeaderSize
	if oidx.header.ocnt&0x7f == 0 {
		// need new page
		if e := oidx.extendBy(objectsPageSize); e != nil {
			return e
		}
		oidx.header.pcnt++
	}
	n := copy(oidx.buf[offset:], oid)
	if n != objectsRecordSize {
		panic(errors.Fault("n is %d!", n))
	}
	oidx.header.ocnt++
	if !oidx.modified {
		oidx.modified = true
	}
	return nil
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

/// oidx internals /////////////////////////////////////////////////////////////

// oidxFile.mmap mmaps the source file. This may be a re-map operation if buf
// is not nil and remap is explicitly requested. buf != nil and remap == false
// is considered a Bug. Otherwise, this is the first mapping.
//
// Returns
func (oidx *oidxFile) mmap(offset int64, length int, remap bool) error {
	if oidx.buf != nil {
		if !remap {
			return errors.Bug("oidxFile.mmap: mmap with existing mapping - remap: %t", remap)
		}
		if e := oidx.unmap(); e != nil {
			return e
		}
	}
	var fd = int(oidx.file.Fd())
	buf, e := syscall.Mmap(fd, offset, length, oidx.prot, oidx.flags)
	if e != nil {
		return e
	}
	// Note: unmap on any errors below

	oidx.buf = buf
	oidx.offset = offset
	// REVU modified should be reset here, no?
	if !remap {
		if e := oidx.header.decode(oidx.buf); e != nil {
			oidx.unmap()
			return e
		}
	}

	return nil
}

func (oidx *oidxFile) unmap() error {
	if oidx.buf == nil {
		return errors.Bug("oidxFile.unmap: buf is nil")
	}
	if oidx.modified {
		return errors.Bug("oidxFile.unmap: modified is true")
	}
	if e := syscall.Munmap(oidx.buf); e != nil {
		return errors.ErrorWithCause(e, "oidxFile.unmap")
	}
	oidx.buf = nil
	oidx.offset = 0
	return nil
}

func (oidx *oidxFile) unmapAndClose() error {
	if oidx.modified {
		oidx.header.updated = time.Now().UnixNano()
		oidx.header.encode(oidx.buf)
	}
	if e := oidx.unmap(); e != nil {
		return errors.ErrorWithCause(e, "oidxFile.unmapAndClose")
	}
	if e := oidx.file.Close(); e != nil {
		return errors.ErrorWithCause(e, "oidxFile.unmapAndClose")
	}
	oidx.header = nil
	oidx.file = nil
	oidx.finfo = nil
	oidx.prot = 0
	oidx.flags = 0
	oidx.opMode = 0 // invalid
	return nil
}

// oidxFile.extendBy extends the file size by delta, and then remaps the
// oidxFile buffer.
func (oidx *oidxFile) extendBy(delta int64) error {

	errorWithCause := errors.ErrorWithCause

	size := oidx.finfo.Size() + delta
	if e := oidx.file.Truncate(size); e != nil {
		fmt.Fprintf(os.Stderr, "debug - error on truncate to %d - will unmap and close file", size)
		oidx.unmapAndClose()
		return errorWithCause(e, "oidxFile.extendBy")
	}
	// update state
	finfo, e := oidx.file.Stat()
	if e != nil {
		fmt.Fprintf(os.Stderr, "debug - error on file.Stat - will unmap and close file")
		oidx.unmapAndClose()
		return errorWithCause(e, "oidxFile.extendBy")
	}
	oidx.finfo = finfo
	// remap
	if e := oidx.mmap(0, int(oidx.finfo.Size()), true); e != nil {
		fmt.Fprintf(os.Stderr, "debug - error on remap - will unmap and close file")
		oidx.unmapAndClose()
		return errorWithCause(e, "oidxFile.extendBy")
	}

	return nil
}

// CreateIndex will create the objects.idx file in the <home>/index/ directory.
// Error is returned if in-arg home is zerolen or if the file already exists.
// The initial index file is simply the header.
/// op mode ////////////////////////////////////////////////////////////////////

// OpMode is a flag type indicating index file access modes. It is used by
// tagmaps and object-index files.
type OpMode byte

const (
	Read OpMode = 1 << iota
	Write
	Verify
	Compact
)

// panics on invalid opMode
func (m OpMode) verify() error {
	switch m {
	case Read:
	case Write:
	case Verify:
	case Compact:
	default:
		return errors.Bug("index.OpMode: unknown mode - %d", m)
	}
	return nil
}

// Returns string rep. of opMode
func (m OpMode) String() string {
	switch m {
	case Read:
		return "Read"
	case Write:
		return "Write"
	case Verify:
		return "Verify"
	case Compact:
		return "Compact"
	}
	panic(errors.Bug("index.OpMode: unknown mode - %d", m))
}
