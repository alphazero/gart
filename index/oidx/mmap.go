// Doost!

package oidx

import (
	"fmt"
	"os"
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

/// memory mapped object index file ////////////////////////////////////////////

const headerSize = 0x1000
const pageSize = 0x1000
const recordSize = index.OidSize // expecting 32 - TODO assert this in init

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
	var offset = ((mmf.header.ocnt) << 5) + headerSize
	if mmf.header.ocnt&0x7f == 0 {
		// need new page
		if e := mmf.ExtendBy(pageSize); e != nil {
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

func (mmf *mappedFile) ExtendBy(delta int64) error {

	size := mmf.finfo.Size() + delta
	if e := mmf.file.Truncate(size); e != nil {
		fmt.Printf("error on truncate to %d - will unmap and close file", size)
		mmf.UnmapClose()
		return e
	}
	// update state
	finfo, e := mmf.file.Stat()
	if e != nil {
		fmt.Printf("error on file.Stat - will unmap and close file")
		mmf.UnmapClose()
		return e
	}
	mmf.finfo = finfo
	// remap
	if e := mmf.mmap(0, int(mmf.finfo.Size()), true); e != nil {
		fmt.Printf("error on remap - will unmap and close file")
		mmf.UnmapClose()
		return e
	}

	return nil
}

func (mmf *mappedFile) CloseIndex() (bool, error) {
	panic("mappedFile.CloseIndex: not implemented.")
}

// REVU not exported
// TODO mmf.Close() (bool, error) calls this
func (mmf *mappedFile) UnmapClose() error {
	// NOTE the write of header is here is very important
	// REVU possibly should be moved to Extend directly?
	// TODO first try it in ref/mmap.go
	if mmf.modified {
		mmf.header.updated = time.Now().UnixNano()
		writeHeader(mmf.header, mmf.buf)
		// REVU this should return a bool indicating header update
	}
	if e := mmf.Unmap(); e != nil {
		return e
	}
	if e := mmf.file.Close(); e != nil {
		return e
	}
	mmf.file = nil
	return nil
}

// if mmf.buf != nil, function first unmaps and then maps again at given
// offset and for buf length specfieid -- effectively remap. Otherwise it
// just a mapping.
func (mmf *mappedFile) mmap(offset int64, length int, remap bool) error {
	if mmf.buf != nil {
		if !remap {
			return fmt.Errorf("mmap with existing mapping - remap: %t", remap)
		}
		if e := mmf.Unmap(); e != nil {
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

// REVU not exported
func (mmf *mappedFile) Unmap() error {
	if mmf.buf == nil {
		return fmt.Errorf("mappedFile.Unmap: buf is nil")
	}
	if e := syscall.Munmap(mmf.buf); e != nil {
		return fmt.Errorf("mappedFile.Unmap: %v", e)
	}
	mmf.buf = nil
	mmf.offset = 0
	return nil
}

// NOTE OpenIndex:
func mapfile(opMode OpMode, filename string) (*mappedFile, error) {
	var flags int // = syscall.MAP_SHARED // syscall.MAP_PRIVATE
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
		panic("bug - invalid opMode")
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
