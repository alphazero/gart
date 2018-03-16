// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/unixtime"
)

var _ = fs.OpenNewFile

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	idx_file_code = 0x763f079cf73c668e // sha256("index-file")[:8]
	idxFilename   = "object.idx"       // REVU belongs to toplevle gart package
)

/// object.idx file header /////////////////////////////////////////////////////

// REVU this should be in gart/system.go

// file header
type idxfile_header struct {
	ftype    uint64
	crc64    uint64        // crc of header bytes from created.
	created  unixtime.Time // unsigned 32bits
	updated  unixtime.Time // unsigned 32bits
	revision uint64        // 0 is new
	rcnt     uint64        // number of records includes those marked for deletion, etc.
	ocnt     uint64        // number of object records. ocnt <= rnct
	reserved [4048]byte    // reserved XXX fix size
}

func init() {
	var hdr idxfile_header
	if unsafe.Sizeof(hdr) != idxfileHeaderBytes {
		panic(fmt.Sprintf("assert fail - idxfile_header size:%d\n", unsafe.Sizeof(hdr)))
	}
}

const idxfileHeaderBytes = 4096

func (h *idxfile_header) writeTo(w io.Writer) error {

	var buf [idxfileHeaderBytes]byte

	*(*uint64)(unsafe.Pointer(&buf[0])) = h.ftype
	*(*uint32)(unsafe.Pointer(&buf[16])) = h.created.Timestamp()
	*(*uint32)(unsafe.Pointer(&buf[20])) = h.updated.Timestamp()
	*(*uint64)(unsafe.Pointer(&buf[24])) = h.revision
	*(*uint64)(unsafe.Pointer(&buf[32])) = h.rcnt
	*(*uint64)(unsafe.Pointer(&buf[40])) = h.ocnt

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64

	_, e := w.Write(buf[:])
	return e
}

func (idx *idxfile) readAndVerifyHeader() error {

	var buf = make([]byte, idxfileHeaderBytes)

	_, e := idx.file.Seek(0, os.SEEK_SET)
	if e != nil {
		return e
	}
	var n int
	for n < len(buf) {
		n0, e := idx.file.Read(buf[n:])
		if e != nil {
			return fmt.Errorf("idxfile.readAndVerifyHeader: Read - n: %d - %s", n, e)
		}
		n += n0
	}

	var h = *(*idxfile_header)(unsafe.Pointer(&buf[0]))

	crc64 := digest.Checksum64(buf[16:])
	if h.crc64 != crc64 {
		return fmt.Errorf("idxfile.readAndVerifyHeader: crc - read:%d computed:%d", h.crc64, crc64)
	}

	(*idx).idxfile_header = h
	return nil
}

/// object.idx file ////////////////////////////////////////////////////////////

// object.idx memory model
type idxfile struct {
	idxfile_header
	opMode   idxOpMode
	file     *os.File
	filename string
	offset   uint64
	modified bool
}

// object.idx file record fixed-width prefix
type idxrec_header struct {
	flags   byte              // deleted, updated, what else?
	oid     [oidBytesLen]byte // ref. to index.idx.file record
	tbahlen uint8
	sbahlen uint8
}

// object.idx file record
type idx_record struct {
	header    idxrec_header
	tags      bitmap.Bitmap
	systemics bitmap.Bitmap
	date      unixtime.Time
}

/// ops ////////////////////////////////////////////////////////////////////////

var (
	ErrIdxOpMode = fmt.Errorf("object.idx: illegal state - idxOpModeMode")
)

type idxOpMode int

// object.idx op/access modes
const (
	IdxCreate idxOpMode = 1 << iota
	IdxRead
	IdxUpdate
	IdxCompact
)

func IdxFilename(garthome string) string {
	if garthome == "" {
		panic("bug - index.Idxfilename: garthome is zerolen")
	}
	// REVU both "index" and idxFileName should be in package gart
	return filepath.Join(garthome, "index", idxFilename)
}

// init - used by gart-init
// create the minimal object.idx file:
// idxfile_header and buflen
// Creates, initializes, and then closes the object.idx file in the
// specified gart dir.
func CreateIdxFile(garthome string) error {

	var filename = IdxFilename(garthome)
	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return fmt.Errorf("index.createIdxFile: %s", e)
	}
	defer file.Close()

	var header = idxfile_header{
		ftype:   idx_file_code,
		crc64:   0,
		created: unixtime.Now(),
	}

	if e := header.writeTo(file); e != nil {
		return fmt.Errorf("index.createIdxFile: %s", e)
	}

	return nil
}

// open
//	gart-add, gart-tag:     os.O_RDWR   flag
//  gart-find, [gart-list]: os.O_RDONLY flag
func OpenIdxFile(garthome string, opMode idxOpMode) (*idxfile, error) {

	var flag int

	switch opMode {
	case IdxRead:
		flag = os.O_RDONLY
	case IdxUpdate:
		flag = os.O_RDWR | os.O_SYNC
	case IdxCompact:
		flag = os.O_RDWR | os.O_SYNC
		panic("idxfile openIdxFile: mode:IdxCompact not implemented")
	default:
		panic(fmt.Errorf("bug - index.openIdxFile: invalid opMode: %d", opMode))
	}

	var filename = IdxFilename(garthome)
	file, e := os.OpenFile(filename, flag, fs.FilePerm)
	if e != nil {
		return nil, fmt.Errorf("index.openIdxFile: %s", e)
	}

	idx := &idxfile{
		opMode:   opMode,
		file:     file,
		filename: filename,
	}

	if e := idx.readAndVerifyHeader(); e != nil {
		idx.file.Close()
		return nil, e
	}

	fmt.Printf("debug - idxfile after read header:\n%v\n", idx)

	idx.file.Close() // XXX TEMP XXX
	panic("idxfile openIdxFile: not implemented")
}

// add object - gart-add - oflag must be os.O_RDWR
// append idx record and return offset
func (f *idxfile) Add(oid *OID, tags, systemics bitmap.Bitmap, date unixtime.Time) (uint64, error) {
	// assert state
	if f.opMode != IdxUpdate {
		return notIndexed, ErrIdxOpMode
	}

	// create new record
	var header = idxrec_header{
		flags:   0,
		oid:     oid.dat, // REVU we're writing to file so no need to copy
		tbahlen: uint8(len(tags.Bytes())),
		sbahlen: uint8(len(systemics.Bytes())),
	}

	var record = idx_record{
		header:    header,
		tags:      tags,
		systemics: systemics,
		date:      date,
	}

	// seek end, write record, get new offset
	roff, e := f.file.Seek(0, os.SEEK_END)
	if e != nil {
		return notIndexed, e
	}

	n, e := record.writeTo(f.file)
	if e != nil {
		return notIndexed, e
	}

	f.offset = uint64(roff + int64(n)) // REVU so do we even need this field? (for read?)
	f.onUpdate()

	// we don't sync
	return uint64(roff), nil
}

// update object - gart-add, gart-tag, (gart-compact?) - oflag must be O_Update
//func (f *idxfile) Update(card Card) (error) // REVU interesting ..
func (f *idxfile) update(roff int64, oid *OID, tags, systemics bitmap.Bitmap) (uint64, error) {
	panic("idxfile.update: not implemented")
}

func (f *idx_record) writeTo(w io.Writer) (int, error) {
	panic("idx_record.writeTo: not implemented")
}

func (f *idxfile) onUpdate() {
	f.updated = unixtime.Now()
	if !f.modified {
		f.modified = true
		f.revision++
	}
}

func (idx *idxfile) DebugStr() string { return fmt.Sprintf("debug: %v", idx) }
