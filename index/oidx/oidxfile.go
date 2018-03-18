// Doost!

package oidx

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
)

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	idx_file_code = 0x763f079cf73c668e // sha256("index-file")[:8]
	idxFilename   = "object.idx"       // REVU belongs to toplevle gart package
)

const block_header_size = 8

// object.idx file consists of 1 or more blocks. Each block is prefixed with
// by an 8 byte header and the rest of the block is a sequence of OIDs.
type block_header struct {
	crc32 uint32
	rcnt  uint32 // number of records in the block
}

// 32KB blocks
const (
	blockSize       = 32768
	blockHeaderSize = 32
	blockDataSize   = 32736 // 1023 32byte object hashes
	blockRecordSize = 32
	recordsPerBlock = 1023
)

// file consists of header and 0 or more blocks. blocks are multiples of fs
// pagesize.
type block struct {
	crc64    uint64
	created  int64  // std unix nano
	updated  int64  // std unix nano
	rcnt     uint32 // number of records in the block
	reserved [4]byte
	dat      [blockDataSize]byte
}

const headerSize = 4096 // file header is fs page sized
type header struct {
	ftype    uint64
	crc64    uint64 // header crc
	created  int64
	updated  int64
	bcnt     uint64 // block count
	rcnt     uint64 // record count
	reserved [4048]byte
}

const recordSize = digest.HashBytes // assert this on init
type idxfile struct {
	header
	file     *os.File
	filename string
	size     int64
	modified bool
	pending  *block
	nextkey  uint64
}

// panics on zerolen input REVU index pkg should give it the full name
func Filename(home string) string {
	if home == "" {
		panic("bug - oidx.idxfilename: garthome is zerolen")
	}
	return filepath.Join(home, "index", "objects.idx")
}

/// errors ////////////////////////////////////////////////////////////////////

var (
	ErrObjectNotFound = fmt.Errorf("object.idx: OID for key not found")
	ErrInvalidOid     = fmt.Errorf("object.idx: Invalid OID")
)

// Creates file, writes initial header and closes file.
func CreateIndex(home string) error {
	var filename = Filename(home)

	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
	}
	defer file.Close()

	var now = time.Now().UnixNano()
	var hdr = header{
		ftype:   idx_file_code,
		created: now,
		updated: now,
		bcnt:    0,
		rcnt:    0,
	}

	var buf [headerSize]byte
	hdr.encode(buf[:])

	_, e = file.Write(buf[:])
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
	}

	fmt.Printf("debug - oidx.CreateIndex:\n%v\n", hdr)
	return nil
}

func (h *header) encode(buf []byte) {
	*(*uint64)(unsafe.Pointer(&buf[0])) = h.ftype
	*(*int64)(unsafe.Pointer(&buf[16])) = h.created
	*(*int64)(unsafe.Pointer(&buf[24])) = h.updated
	*(*uint64)(unsafe.Pointer(&buf[32])) = h.bcnt
	*(*uint64)(unsafe.Pointer(&buf[40])) = h.rcnt

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64

	return
}

func (idx *idxfile) readAndVerifyHeader() error {

	_, e := idx.file.Seek(0, os.SEEK_SET)
	if e != nil {
		return e
	}

	var buf = make([]byte, headerSize)
	var n int
	for n < len(buf) {
		n0, e := idx.file.Read(buf[n:])
		if e != nil {
			return fmt.Errorf("idxfile.readAndVerifyHeader: Read - n: %d - %s", n, e)
		}
		n += n0
	}

	(*idx).header = *(*header)(unsafe.Pointer(&buf[0]))

	crc64 := digest.Checksum64(buf[16:])
	if idx.crc64 != crc64 {
		return fmt.Errorf("idxfile.readAndVerifyHeader: crc:%d computed:%d", idx.crc64, crc64)
	}
	return nil
}

// Opens the object index file. REVU mode?
func OpenIndex(home string) (*idxfile, error) {
	var filename = Filename(home)

	// open file and get stat
	file, e := fs.OpenNewFile(filename, os.O_RDWR)
	if e != nil {
		return nil, fmt.Errorf("oidx.OpenIndex: %s", e)
	}
	finfo, e := file.Stat()
	if e != nil {
		return nil, fmt.Errorf("oidx.OpenIndex: unexpected: %s", e)
	}

	// initialize idxfile
	idx := &idxfile{
		file:     file,
		filename: filename,
		size:     finfo.Size(),
		modified: false,
		pending:  nil,
	}

	// read header and verify
	if e := idx.readAndVerifyHeader(); e != nil {
		idx.file.Close()
		return nil, e
	}

	return idx, nil
}

func (idx *idxfile) Register(oid []byte) (uint64, error) {
	if oid == nil || len(oid) != recordSize {
		return 0, ErrInvalidOid
	}

	panic("oidx.Register: not implemented")
}

func (idx *idxfile) Lookup(key ...uint64) ([][]byte, error) {
	panic("oidx.Lookup: not implemented")
}

func (idx *idxfile) Sync() (bool, error) {
	panic("oidx.Sync: not implemented")
}

func (idx *idxfile) Close() error {
	panic("oidx.Close: not implemented")
}
