// Doost!

package oidx

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/lang/sort"
)

func init() {
	if index.OidSize != 32 {
		panic("package oidx: index.OidSize is not 32")
	}
}

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	idx_file_code = 0x763f079cf73c668e // sha256("index-file")[:8]
	idxFilename   = "object.idx"       // REVU belongs to toplevle gart package
)

// error codes
var (
	ErrInvalidOp        = fmt.Errorf("object.idx: Invalid op for index opMode")
	ErrOpNotImplemented = fmt.Errorf("object.idx: Operation not implemented")
	ErrObjectNotFound   = fmt.Errorf("object.idx: OID for key not found")
	ErrInvalidOid       = fmt.Errorf("object.idx: Invalid OID")
	ErrInvalidKeyRange  = fmt.Errorf("object.idx: Invalid key range")
	ErrNilArg           = fmt.Errorf("object.idx: Invalid arg - nil")
	ErrIndexIsClosed    = fmt.Errorf("object.idx: Invalid state - index already closed")
	ErrPendingChanges   = fmt.Errorf("object.idx: Invalid state - pending changes on close")
)

const (
	headerSize = 0x1000
	pageSize   = 0x1000
	recordSize = index.OidSize
)

/// op mode ///////////////////////////////////////////////////////////////////

type OpMode byte

const (
	Read OpMode = 1 << iota
	Write
	Verify
	Compact
)

// panics on unimplemented op mode
func (m OpMode) fopenFlag() int {
	switch m {
	case Read:
		return os.O_RDONLY
	case Write:
		return os.O_RDWR
	case Verify:
	case Compact:
	default:
	}
	panic(fmt.Errorf("bug - oidx.OpMode: not implemented - mode  %d", m))
}

// panics on invalid opMode
func (m OpMode) verify() error {
	switch m {
	case Read:
	case Write:
	case Verify:
	case Compact:
	default:
		return fmt.Errorf("bug - oidx.OpMode: unknown mode - %d", m)
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
	panic(fmt.Errorf("bug - oidx.OpMode: unknown mode - %d", m))
}

/// memory mapped object index file ////////////////////////////////////////////

type header struct {
	ftype    uint64
	crc64    uint64 // header crc
	created  int64
	updated  int64
	pcnt     uint64 // page count	=> pcnt
	ocnt     uint64 // record count => ocnt
	reserved [4048]byte
}

func (hdr *header) Print() {
	fmt.Printf("file type:  %016x\n", hdr.ftype)
	fmt.Printf("crc64:      %016x\n", hdr.crc64)
	fmt.Printf("created:    %016x (%s)\n", hdr.created, time.Unix(0, hdr.created))
	fmt.Printf("updated:    %016x (%s)\n", hdr.updated, time.Unix(0, hdr.updated))
	fmt.Printf("page cnt: : %d\n", hdr.pcnt)
	fmt.Printf("object cnt: %d\n", hdr.ocnt)
}

func writeHeader(h *header, buf []byte) {
	*(*uint64)(unsafe.Pointer(&buf[0])) = h.ftype
	*(*int64)(unsafe.Pointer(&buf[16])) = h.created
	*(*int64)(unsafe.Pointer(&buf[24])) = h.updated
	*(*uint64)(unsafe.Pointer(&buf[32])) = h.pcnt
	*(*uint64)(unsafe.Pointer(&buf[40])) = h.ocnt

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64

	return
}

func readHeader(buf []byte) *header {
	var h header = *(*header)(unsafe.Pointer(&buf[0]))
	return &h
}

type mappedFile struct {
	*header
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

// XXX temp
func (mmf *mappedFile) DevDebug() {
	if mmf.header.pcnt > 0 {
		mmf.Hexdump(mmf.header.pcnt)
	}
	mmf.header.Print()
}

func (mmf *mappedFile) Lookup(key ...uint64) ([][]byte, error) {
	// assert mode, sort keys, and verify key range validity
	if mmf.opMode != Read {
		return nil, ErrInvalidOp
	}
	if key == nil {
		return nil, ErrNilArg
	}
	var klen = len(key)
	if klen == 0 {
		return [][]byte{}, nil // nop
	}
	sort.Uint64(key)
	if key[0] == 0 || key[klen-1] >= mmf.ocnt {
		return nil, ErrInvalidKeyRange
	}

	var oids = make([][]byte, klen)
	for i, k := range key {
		offset := (k << 5) + headerSize
		oids[i] = make([]byte, recordSize)
		copy(oids[i], mmf.buf[offset:offset+recordSize])
	}

	return oids, nil
}

func (mmf *mappedFile) AddObject(oid []byte) error {
	if mmf.opMode != Write {
		return ErrInvalidOp
	}

	var offset = ((mmf.header.ocnt) << 5) + headerSize
	if mmf.header.ocnt&0x7f == 0 {
		// need new page
		if e := mmf.extendBy(pageSize); e != nil {
			return e
		}
		mmf.header.pcnt++
	}
	n := copy(mmf.buf[offset:], oid)
	if n != recordSize {
		panic(fmt.Errorf("n is %d!", n))
	}
	mmf.header.ocnt++
	if !mmf.modified {
		mmf.modified = true
	}
	return nil
}

func (mmf *mappedFile) Hexdump(page uint64) {
	var p = int(page)
	var width = 16
	var poff = p << 12
	var pend = poff + 0x1000 // page size 4K
	for i := poff; i < pend; i += width {
		fmt.Printf("%08x % 02x\n", i, mmf.buf[i:i+width])
	}
}

func (mmf *mappedFile) extendBy(delta int64) error {

	size := mmf.finfo.Size() + delta
	if e := mmf.file.Truncate(size); e != nil {
		fmt.Printf("error on truncate to %d - will unmap and close file", size)
		mmf.unmapAndClose()
		return e
	}
	// update state
	finfo, e := mmf.file.Stat()
	if e != nil {
		fmt.Printf("error on file.Stat - will unmap and close file")
		mmf.unmapAndClose()
		return e
	}
	mmf.finfo = finfo
	// remap
	if e := mmf.mmap(0, int(mmf.finfo.Size()), true); e != nil {
		fmt.Printf("error on remap - will unmap and close file")
		mmf.unmapAndClose()
		return e
	}

	return nil
}

func filename(home string) string {
	if home == "" {
		panic("bug - oidx.idxfilename: garthome is zerolen")
	}
	return filepath.Join(home, "index", "objects.idx")
}

// CreateIndex will create the objects.idx file in the <home>/index/ directory.
// Error is returned if in-arg home is zerolen or if the file already exists.
// The initial index file is simply the header.
func CreateIndex(home string) error {
	var filename = filename(home)

	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
	}
	defer file.Close()

	var now = time.Now().UnixNano()
	var hdr = &header{
		ftype:   idx_file_code, // TODO uniformly call these _typecode
		created: now,           // TODO REVU timestamps on all files
		updated: now,           // TODO uniformly set updated to created on init on all files
		pcnt:    0,
		ocnt:    0,
	}

	var buf [headerSize]byte
	writeHeader(hdr, buf[:])

	_, e = file.Write(buf[:])
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
	}

	return nil
}

func OpenIndex(home string, opMode OpMode) (*mappedFile, error) {

	if e := opMode.verify(); e != nil {
		return nil, e
	}

	var filename = filename(home)
	var flags int
	var oflags int
	var prot int
	switch opMode {
	case Read:
		prot = syscall.MAP_PRIVATE
		prot = syscall.PROT_READ
		oflags = os.O_RDONLY
	case Write:
		flags = syscall.MAP_SHARED
		prot = syscall.PROT_WRITE
		oflags = os.O_RDWR
	default:
		panic("bug - unsupported opMode")
	}

	file, e := os.OpenFile(filename, oflags, fs.FilePerm)
	if e != nil {
		return nil, e
	}
	// Note:close file on all errors after this point

	finfo, e := file.Stat()
	if e != nil {
		file.Close()
		return nil, e
	}

	mmf := &mappedFile{
		filename: filename,
		opMode:   opMode,
		file:     file,
		finfo:    finfo,
		prot:     prot,
		flags:    flags,
	}

	var offset int64 = 0
	if e := mmf.mmap(offset, int(finfo.Size()), false); e != nil {
		file.Close()
		return nil, e
	}

	return mmf, nil
}
func (mmf *mappedFile) CloseIndex() (bool, error) {
	return mmf.unmapAndClose()
}

// unmapAndClose first updates header timestamp and (re)writes
// the header bytes (if opMode is Read & modified). If so, the
// bool retval will reflect that.
func (mmf *mappedFile) unmapAndClose() (bool, error) {
	// NOTE the write of header is here is very important
	// REVU this should be done in AddObject (which is the only
	// place that actually modifies the header! It does not belong here.
	// TODO first try it in ref/mmap.go
	var updated bool
	if mmf.modified {
		mmf.header.updated = time.Now().UnixNano()
		writeHeader(mmf.header, mmf.buf)
		updated = true
	}
	if e := mmf.unmap(); e != nil {
		return updated, e
	}
	if e := mmf.file.Close(); e != nil {
		return updated, e
	}
	mmf.file = nil
	return updated, nil
}

// if mmf.buf != nil, function first unmaps and then maps again at given
// offset and for buf length specfieid -- effectively remap. Otherwise it
// just a mapping.
func (mmf *mappedFile) mmap(offset int64, length int, remap bool) error {
	if mmf.buf != nil {
		if !remap {
			return fmt.Errorf("mmap with existing mapping - remap: %t", remap)
		}
		if e := mmf.unmap(); e != nil {
			return e
		}
	}
	var fd = int(mmf.file.Fd())
	buf, e := syscall.Mmap(fd, offset, length, mmf.prot, mmf.flags)
	if e != nil {
		return e
	}
	mmf.buf = buf
	mmf.offset = offset
	if !remap {
		mmf.header = readHeader(mmf.buf)
	}

	return nil
}

func (mmf *mappedFile) unmap() error {
	if mmf.buf == nil {
		return fmt.Errorf("mappedFile.unmap: buf is nil")
	}
	if e := syscall.Munmap(mmf.buf); e != nil {
		return fmt.Errorf("mappedFile.unmap: %v", e)
	}
	mmf.buf = nil
	mmf.offset = 0
	return nil
}
