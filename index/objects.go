// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"sort"
	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/repo"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
)

// objects.idx file is a sequential list of object ids. The adjusted offset of
// the fixed witdth Oid data (32B) is the implicit 'key' for the object. The
// adjustment is accounting for the objects.idx objectsHeader.
//
// On creation of new objects, an Oid entry is appended to the objects.idx file.
// The corresponding 'key' is recorded in the corresponding index.Card.
//
// On queries of objects for a given specification of tags (e.g AND or more
// selective logical expressions) an array of 'bits' is obtained from the Tagmap
// and these bit (positions) correspond to the 'keys' of objects.idx, from which
// we maps the tagmap.bits -> object.keys -> Oids -> Cards.

/// consts and vars ///////////////////////////////////////////////////////////

const (
	mmap_idx_file_code uint64 = 0x8fe452c6d1f55c66 // sha256("mmaped-index-file")[:8]
)

var oidxFilename string = repo.ObjectIndexPath

// objectsHeader related consts
const (
	objectsHeaderSize = 0x1000
	objectsPageSize   = 0x1000
	objectsRecordSize = system.OidSize
	objectsPerPage    = objectsPageSize / objectsRecordSize
	ocntExtendMask    = objectsPerPage - 1
)

/// objects.idx specific inits /////////////////////////////////////////////////

func init() {
	// verify system size assumptions central to objects.idx file
	if system.OidSize != digest.HashSize {
		panic(errors.Fault("index/objects.go: Oid-Size:%d", system.OidSize))
	}
}

/// objects.idx file objectsHeader /////////////////////////////////////////////

// objects.idx file's header is a page size (4KB) structure and is the minimal object
// index file. The crc64 field is the checksum of the header only. The reserved bits
// are for a projected merkel checksum of actual object index data chunks.
type objectsHeader struct {
	ftype    uint64
	crc64    uint64 // objectsHeader crc
	created  int64
	updated  int64
	pcnt     uint64 // page count - 1..n
	ocnt     int64  // object count - 1..n
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
	*(*int64)(unsafe.Pointer(&buf[40])) = h.ocnt

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64

	return nil
}

func (h *objectsHeader) decode(buf []byte) error {
	var err = errors.For("objectsHeader.decode")

	if len(buf) < objectsHeaderSize {
		return err.Error("insufficient buffer length: %d",
			len(buf))
	}
	*h = *(*objectsHeader)(unsafe.Pointer(&buf[0]))

	/// verify //////////////////////////////////////////////////////

	if h.ftype != mmap_idx_file_code {
		return err.Bug("invalid ftype: %x - expect: %x",
			h.ftype, mmap_idx_file_code)
	}
	crc64 := digest.Checksum64(buf[16:])
	if crc64 != h.crc64 {
		return err.Bug("invalid checksum: %x - expect: %x",
			h.crc64, crc64)
	}
	if h.created == 0 {
		return err.Bug("invalid created: %d", h.created)
	}
	if h.updated < h.created {
		return err.Bug("invalid updated: %d < created:%d",
			h.updated, h.created)
	}

	return nil
}

/// objects.idx file ///////////////////////////////////////////////////////////

// oidxFile structure captures the persistent and run-time meta-data and data of
// the objects.idx file. The file is memory mapped and supports distinct opModes.
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
	oidx.header.Print(w)
	fmt.Fprintf(w, "---------------------\n")
	fmt.Fprintf(w, "opMode:     %s\n", oidx.opMode)
	fmt.Fprintf(w, "source:     %q\n", oidx.source)
	fmt.Fprintf(w, "buf-len:    %d\n", len(oidx.buf))
	fmt.Fprintf(w, "offset:     %d\n", oidx.offset)
	fmt.Fprintf(w, "modified:   %t\n", oidx.modified)
	if oidx.header.pcnt > 0 {
		fmt.Fprintf(w, "-page: 1   ----------\n")
		oidx.hexdump(w, 1)
	}
	if oidx.header.pcnt > 1 {
		fmt.Fprintf(w, "...                  \n")
		fmt.Fprintf(w, "-page: %2d ----------\n", oidx.header.pcnt)
		oidx.hexdump(w, oidx.header.pcnt)
	}
}

func (oidx *oidxFile) hexdump(w io.Writer, page uint64) {
	var p = int(page)
	var width = 16
	var poff = p << 12
	var pend = poff + 0x1000 // page size 4K
	var lim = int(((oidx.header.ocnt) << 5) + objectsHeaderSize)
	var truncated bool = true
	if lim > pend {
		lim = pend
		truncated = false
	}
	for i := poff; i < lim; i += width {
		fmt.Fprintf(w, "%08x % 02x\n", i, oidx.buf[i:i+width])
	}
	if truncated {
		var i = lim
		fmt.Fprintf(w, "%08x % 02x\n", i, oidx.buf[i:i+width])
		fmt.Fprintf(w, " ...\n")
		i = pend - width
		fmt.Fprintf(w, "%08x % 02x\n", i, oidx.buf[i:i+width])
	}
}

// CreateObjectIndex creates the initial (header only/empty) objects.idx file.
// The file is closed on return.
//
// Returns index.ErrObjectIndexExists if index file already exists.
func createObjectIndex() error {
	var err = errors.For("index.createObjectIndex")

	// create object index file
	file, e := fs.OpenNewFile(oidxFilename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return e
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
			panic(err.Fault("os.Remove - %s - recovering from: %s", ec, e))
		}
		return e
	}

	_, e = file.Write(buf[:])
	if e != nil {
		return e
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

	var debug = debug.For("index.openObjectIndex")
	debug.Printf("called - opMode:%s", opMode)

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

	file, e := os.OpenFile(oidxFilename, oflags, repo.FilePerm)
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
		header: &objectsHeader{},
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

	//	oidx.header.Print(os.Stdout)

	return oidx, nil
}

func (oidx *oidxFile) nextKey() int64 { return oidx.header.ocnt }

// addObject appends the given oid to the object index. Index must have been
// openned in OpMode#Write. The underlying file will be extended by a page
// if required.
//
// Returns the 'key' of the new object, and nil on success.
// On error, the uint64 value should be ignored.
func (oidx *oidxFile) addObject(oid *system.Oid) (int64, error) {
	var err = errors.For("oidxFile.addObject")

	var key int64 = -1
	if oidx.opMode != Write {
		return key, err.Bug("invalid op-mode:%s", oidx.opMode)
	}

	if oidx.header.ocnt&ocntExtendMask == 0 {
		// need new page
		if e := oidx.extendBy(objectsPageSize); e != nil {
			return key, e
		}
		oidx.header.pcnt++
	}
	var offset = ((oidx.header.ocnt) << 5) + objectsHeaderSize
	if e := oid.Encode(oidx.buf[offset:]); e != nil {
		panic(err.FaultWithCause(e,
			"oid:%s - offset:%d - buflen:%d", oid, offset, len(oidx.buf)))
	}

	key = oidx.header.ocnt // REVU this starts keys with 0
	oidx.header.ocnt++
	if !oidx.modified {
		oidx.modified = true
	}
	return key, nil
}

// validateQueryArgs asserts requirements for a Read op of object index,
// including the validity of the input keys.
//
// Returns the sorted keys and nil error if query is valid.
//
// Returns nil, Bug if no keys are specified.
// Returns nil, Bug for key values < 0 or > oidx.object count.
// Returns nil, index.ErrObjectIndexClosed or any other encountered error.
func (oidx *oidxFile) validateQueryArgs(keys ...int) ([]int, error) {
	var err = errors.For("oidxFile.validateQueryArgs")
	if oidx.opMode != Read {
		return nil, err.Bug("invalid op-mode:%s", oidx.opMode)
	}
	if len(keys) == 0 { // a bug given that oidx (self) is the (only) caller.
		panic("here")
		return nil, err.Bug("no keys specified")
	}

	sort.Ints(keys)
	if k := keys[0]; k < 0 {
		return nil, err.Bug("invalid key: %d", k)
	}
	if k := keys[len(keys)-1]; k >= int(oidx.header.ocnt) {
		return nil, err.Bug("invalid key: %d", k)
	}
	return keys, nil
}

// getOids returns the oids for the provided keys.
//
// Returns the oids corresponding to the sorted key set.
//
// Returns nil, Bug if no keys are specified.
// Returns nil, Bug for key values < 0 or > oidx.object count.
// Returns nil, index.ErrObjectIndexClosed or any other encountered error.
func (oidx *oidxFile) getOids(keys ...int) ([]*system.Oid, error) {
	if oidx.file == nil {
		return nil, ErrObjectIndexClosed
	}
	if len(keys) == 0 {
		return []*system.Oid{}, nil
	}
	keys, e := oidx.validateQueryArgs(keys...)
	if e != nil {
		return nil, e
	}
	var oids = make([]*system.Oid, len(keys))
	for i, k := range keys {
		offset := (k << 5) + objectsHeaderSize
		oid, e := system.NewOid(oidx.buf[offset : offset+objectsRecordSize])
		if e != nil {
			return nil, e
		}
		oids[i] = oid
	}

	return oids, nil
}

// getOids returns all oids excluding those mapped by the provided keys.
//
// Returns nil, Bug if no keys are specified.
// Returns nil, Bug for key values < 0 or > oidx.object count.
//
// Returns index.ErrObjectIndexClosed or any other encountered error, in which
// the resultant map will be nil.
func (oidx *oidxFile) getOidsExcluding(keys ...int) ([]*system.Oid, error) {
	if len(keys) == 0 {
		return []*system.Oid{}, nil
	}
	keys, e := oidx.validateQueryArgs(keys...)
	if e != nil {
		return nil, e
	}

	// keys is (now) a sorted array with first element >=0 and last element
	// < max oidx key.
	var ocnt = int(oidx.header.ocnt)
	var oids = make([]*system.Oid, ocnt-len(keys))
	var n int // indexes oids
	var j int // indexes keys
next_key:
	for k := 0; k < ocnt; k++ {
		for j < len(keys) && keys[j] < k {
			if keys[j] == k {
				continue next_key
			}
			j++
		}
		offset := (k << 5) + objectsHeaderSize
		oid, e := system.NewOid(oidx.buf[offset : offset+objectsRecordSize])
		if e != nil {
			return nil, e
		}
		oids[n] = oid
		n++
	}
	return oids, nil
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

	// read header on initial mapping
	// header is updated to file only on closeIndex.
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
	if e := syscall.Munmap(oidx.buf); e != nil {
		return errors.ErrorWithCause(e, "oidxFile.unmap")
	}
	oidx.buf = nil
	oidx.offset = 0
	return nil
}

func (oidx *oidxFile) extendBy(delta int64) error {
	var err = errors.For("oidxFile.extendBy")

	size := oidx.finfo.Size() + delta
	if e := oidx.file.Truncate(size); e != nil {
		oidx.closeIndex(false)
		return err.Error("on truncate to size:%d - %v", size, e)
	}
	// update state
	finfo, e := oidx.file.Stat()
	if e != nil {
		oidx.closeIndex(false)
		return e
	}
	oidx.finfo = finfo
	// remap
	if e := oidx.mmap(0, int(oidx.finfo.Size()), true); e != nil {
		oidx.closeIndex(false)
		return err.Error("on oidx.mmap - %v", e)
	}

	return nil
}

// closeIndex closes the index, at which point the reference to the pointer
// should be discarded. On commit, if modified, header is reencoded to the mapped
// buffer before unmapping and closing the file. (This is because the header is
// copied from mapped buffer on open/mmap and updates to header fields are not
// direclty recorded on the mapped buffer.) If commit is false, we re-read the
// header directly, trunc the file back to the original size, and remove any
// residual records in the last page.
//
// Returns index.ErrObjectIndexClosed if index has already been closed.
//
// panics on unrecoverable errors.
func (oidx *oidxFile) closeIndex(commit bool) error {
	//func (oidx *oidxFile) unmapAndClose() error {
	var err = errors.For("oidxFile.closeIndex")
	if oidx.modified {
		if commit {
			oidx.header.updated = time.Now().UnixNano()
			if e := oidx.header.encode(oidx.buf); e != nil {
				panic(err.Fault(e.Error()))
			}
			oidx.modified = false // REVU is this necessary?
		} else {
			// reset header from buf
			*(oidx.header) = *(*objectsHeader)(unsafe.Pointer(&oidx.buf[0]))
			// truncate to initial load size
			size := objectsHeaderSize + (oidx.header.pcnt * objectsPageSize)
			if e := oidx.file.Truncate(int64(size)); e != nil {
				panic(err.Fault(e.Error()))
			}
			// clear residual records in last page
			offset := int(((oidx.header.ocnt) << 5) + objectsHeaderSize)
			for xof := offset; xof < int(size); xof++ {
				oidx.buf[xof] = 0
			}
		}
	}
	if e := oidx.unmap(); e != nil {
		return err.ErrorWithCause(e, "oidx.unmap")
	}
	if e := oidx.file.Close(); e != nil {
		return err.ErrorWithCause(e, "file.Close")
	}

	oidx.header = nil
	oidx.file = nil
	oidx.finfo = nil
	oidx.prot = 0
	oidx.flags = 0
	oidx.opMode = 0 // invalid
	return nil
}
