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
	blockDataSize   = 32736
	blockRecordSize = 32
	recordsPerBlock = 1023
)

type block struct {
	crc64    uint64
	created  int64  // std unix nano
	updated  int64  // std unix nano
	rcnt     uint32 // number of records in the block
	reserved [4]byte
	dat      [blockDataSize]byte
}

const headerSize = 4096 // fs page
type header struct {
	ftype    uint64
	crc64    uint64 // header crc
	created  int64
	updated  int64
	bcnt     uint64 // block count
	rcnt     uint64 // record count
	reserved [4048]byte
}

type idxfile struct {
	header
	file     *os.File
	filename string
	size     int64
	modified bool
	pending  *block
	nextkey  uint64
}

// panics on zerolen input
func Filename(home string) string {
	if home == "" {
		panic("bug - oidx.idxfilename: garthome is zerolen")
	}
	return filepath.Join(home, "index", "objects.idx")
}

// Creates file, writes initial header and closes file.
func CreateIndex(home string) error {
	var filename = Filename(home)

	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return fmt.Errorf("index.createIdxFile: %s", e)
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
	*(*uint64)(unsafe.Pointer(&buf[0])) = hdr.ftype
	*(*int64)(unsafe.Pointer(&buf[16])) = hdr.created
	*(*int64)(unsafe.Pointer(&buf[24])) = hdr.updated
	*(*uint64)(unsafe.Pointer(&buf[32])) = hdr.bcnt
	*(*uint64)(unsafe.Pointer(&buf[40])) = hdr.rcnt

	hdr.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = hdr.crc64

	_, e = file.Write(buf[:])
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
	}

	return nil
}
