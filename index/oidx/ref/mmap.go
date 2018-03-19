// Doost!

package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
	//	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/index/oidx"
)

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// to try:
	// mmap a file to its full extent
	// increase the size of the file
	// investigate the process of (re)mapping for the longer extent.
	// Doost!

	// NOTE
	// this works fine and possible optimizations are:
	// 	- increase page size as each call to ExtendTo() incurs cost of unmap & mmap
	//	  for example, 32KB page size extends every 1024 records.
	//	  Likely we're wasting tens of KB.
	//
	//	- don't use pages at all. On OpMode.Write, start with a resize to something
	//	  very large, say inrease by k * MB. Say 4MB (~128k objects to be added) and
	//	  that takes care of most use cases. Then on UnmapClose trunc to actual size.
	//	  That would require more extensive changes, but would certainly be faster.
	//	  REVU this is clearly the optimal approach.(Resize extent can be adjusted
	//	  for host device type to address mobile-phone use-cases.
	//
	//	- But also try MAP_NOCACHE flag on OpWrite to see if that helps.

	var filename = "/Users/alphazero/.gart/index/objects.idx"

	debug := len(os.Args) > 1

	if debug {
		if e := readFromIt(filename); e != nil {
			exitOnError(e)
		}
	}

	if e := writeToIt(filename, 333); e != nil {
		exitOnError(e)
	}

	if debug {
		if e := readFromIt(filename); e != nil {
			exitOnError(e)
		}
	}
}

func writeToIt(filename string, items int) error {
	// gart process prepare:
	mmf, e := mapfile(oidx.Write, filename)
	if e != nil {
		exitOnError(e)
	}

	// gart process complete:
	defer mmf.UnmapClose()

	// gart-add:
	for i := 0; i < items; i++ {
		oid := digest.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		if e := mmf.AddObject(oid[:]); e != nil {
			fmt.Printf("err - writeToIt: %v", e)
			return e
		}
	}

	return nil
}

const headersize = 0x1000
const pagesize = 0x1000
const recsize = 0x20

func (mmf *mappedFile) Lookup(key ...uint64) ([][]byte, error) {
	return nil, fmt.Errorf("not implemented -- use exiting code in oidx.oidxfile.go")
}

func (mmf *mappedFile) AddObject(oid []byte) error {
	var offset = ((mmf.header.ocnt) << 5) + headersize
	if mmf.header.ocnt&0x7f == 0 {
		// need new page
		if e := mmf.ExtendBy(pagesize); e != nil {
			return e
		}
		mmf.header.pcnt++
	}
	n := copy(mmf.buf[offset:], oid)
	if n != 32 {
		panic(fmt.Errorf("n is %d!", n))
	}
	mmf.header.ocnt++
	if !mmf.modified {
		mmf.modified = true
	}
	return nil
}

// REVU deprecated
func (mmf *mappedFile) AddObject_works(oid []byte) error {
	var offset = ((mmf.header.ocnt) << 5) + headersize
	switch mmf.header.ocnt & 0x7f {
	case 0: // need new page
		if e := mmf.ExtendBy(pagesize); e != nil {
			return e
		}
		mmf.header.pcnt++
		//		fmt.Printf("add object to NEW page:%d at offset:%d\n", mmf.header.pcnt, offset)
	default: // partial page
		//		fmt.Printf("add object to page:%d at offset:%d\n", mmf.header.pcnt, offset)
	}
	n := copy(mmf.buf[offset:], oid)
	if n != 32 {
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

type header struct {
	ftype    uint64
	crc64    uint64 // header crc
	created  int64
	updated  int64
	pcnt     uint64 // page count	=> pcnt
	ocnt     uint64 // record count => ocnt
	reserved [4048]byte
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

func (hdr *header) Print() {
	fmt.Printf("file type:  %016x\n", hdr.ftype)
	fmt.Printf("crc64:      %016x\n", hdr.crc64)
	fmt.Printf("created:    %016x (%s)\n", hdr.created, time.Unix(0, hdr.created))
	fmt.Printf("updated:    %016x (%s)\n", hdr.updated, time.Unix(0, hdr.updated))
	fmt.Printf("page cnt: : %d\n", hdr.pcnt)
	fmt.Printf("object cnt: %d\n", hdr.ocnt)
}

func readFromIt(filename string) error {
	mmf, e := mapfile(oidx.Read, filename)
	if e != nil {
		exitOnError(e)
	}
	fmt.Printf("debug - readFromIt - bufsize: %d\n", len(mmf.buf))

	//	hdr := readHeader(mmf.buf)
	mmf.header.Print()
	hdr := mmf.header
	// display last page if any
	if hdr.pcnt > 0 {
		fmt.Printf("last data page\n")
		mmf.Hexdump(hdr.pcnt)
	}
	return mmf.UnmapClose()
}

type mappedFile struct {
	*header
	filename string
	opMode   oidx.OpMode
	file     *os.File
	finfo    os.FileInfo // size is int64
	flags    int
	prot     int
	buf      []byte
	offset   int64
	modified bool
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

func (mmf *mappedFile) UnmapClose() error {
	if mmf.modified {
		mmf.header.updated = time.Now().UnixNano()
		writeHeader(mmf.header, mmf.buf)
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

// OpenIndex:
func mapfile(opMode oidx.OpMode, filename string) (*mappedFile, error) {
	var flags int // = syscall.MAP_SHARED // syscall.MAP_PRIVATE
	var oflags int
	var prot int
	switch opMode {
	case oidx.Read:
		prot = syscall.MAP_PRIVATE
		prot = syscall.PROT_READ
		oflags = os.O_RDONLY
	case oidx.Write:
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

func exitOnError(e error) {
	fmt.Printf("err - %s\n", e)
	os.Exit(1)
}
