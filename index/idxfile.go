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

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	idx_file_code = 0x763f079cf73c668e // sha256("index-file")[:8]
	idxFilename   = "object.idx"       // REVU belongs to toplevle gart package
)

/// object.idx file header /////////////////////////////////////////////////////

// XXX
func init() {
	var hdr idxfile_header
	if unsafe.Sizeof(hdr) != idxfileHeaderBytes {
		panic(fmt.Sprintf("assert fail - idxfile_header size:%d\n", unsafe.Sizeof(hdr)))
	}
	ops := []idxOpMode{
		IdxCreate,
		IdxRead,
		IdxVerify,
		IdxUpdate,
		IdxCompact,
	}
	for _, v := range ops {
		fmt.Printf("op-mode: %08b\n", v)
	}
}

// XXX

/// object.idx file ////////////////////////////////////////////////////////////

const idxfileHeaderBytes = 4096

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

// object.idx memory model
type idxfile struct {
	idxfile_header
	opMode    idxOpMode      // operation mode
	file      *os.File       // file handle - nil after close
	filename  string         // source file
	size      uint64         // file size at read / after sync
	modified  bool           // necessary since mod can be in-place
	appendLog []idxPendingOp // idx_records to be appended
	delset    []uint64       // offset of existing records marked deleted
	modset    []idxPendingOp //
	poff      uint64         // poff: pending (or projected) end offset after sync
}

type idxPendingOp struct {
	offset uint64
	record *idx_record
}

// idx record flag masks
const (
	idxrec_invalid = 0
	idxrec_valid   = 0x80                // 10000000
	idxrec_deleted = idxrec_valid | 0x40 // 11000000
	idxrec_updated = idxrec_valid | 0x20 // 10100000
	idxrec_moved   = idxrec_valid | 0x10 // 10010000
)

// object.idx file record fixed-width prefix
type idxrec_header struct {
	flags   byte              // deleted, updated, what else?
	oid     [oidBytesLen]byte // ref. to index.idx.file record
	tbahlen uint8
	sbahlen uint8
}

// object.idx file record
type idx_record struct {
	idxrec_header               // 3 + oidByteLen
	date          unixtime.Time // 4b
	tags          bitmap.Bitmap // var
	systemics     bitmap.Bitmap // var
}

// Returns the length of the record in bytes.
func (rec *idx_record) length() int {
	return 7 + oidBytesLen + int(rec.tbahlen) + int(rec.sbahlen)
}
func (rec *idx_record) String() string {
	s := fmt.Sprintf("idx-record:(flag:%08b oid:%x tlen:%02d slen:%02d date:%s tags:%08b systemics:%08b", rec.flags, rec.oid, rec.tbahlen, rec.sbahlen, rec.date.Date(), rec.tags, rec.systemics)
	return s
}

/// header codec ///////////////////////////////////////////////////////////////

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

/// ops ////////////////////////////////////////////////////////////////////////

var (
	ErrIdxOpMode         = fmt.Errorf("object.idx: illegal state - idxOpMode")
	ErrIdxClosed         = fmt.Errorf("object.idx: illegal state - closed")
	ErrIdxPendingChanges = fmt.Errorf("object.idx: illegal state - close with pending changes")
)

type idxOpMode byte

// object.idx op/access modes
const (
	IdxCreate  idxOpMode = 1 << iota
	IdxRead              // Read only mode - used for queries
	IdxVerify            // Read only mode - used for system verification
	IdxUpdate            // RW mode - used for gart object updates
	IdxCompact           // RW mode - used for gc'ing and repairs (moved recs etc.)
)

func IdxFilename(garthome string) string {
	if garthome == "" {
		panic("bug - index.Idxfilename: garthome is zerolen")
	}
	// REVU both "index" and idxFileName should be in package gart
	return filepath.Join(garthome, "index", idxFilename)
}

// Creates, initializes, and then closes the object.idx file in the
// specified gart repo.
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
	case IdxVerify:
		flag = os.O_RDONLY
	case IdxUpdate:
	case IdxCompact:
		flag = os.O_RDWR | os.O_SYNC // TODO research this O_SYNC flag
	default:
		panic(fmt.Errorf("bug - index.openIdxFile: invalid opMode: %d", opMode))
	}

	var filename = IdxFilename(garthome)
	file, e := os.OpenFile(filename, flag, fs.FilePerm)
	if e != nil {
		return nil, fmt.Errorf("index.openIdxFile: %s", e)
	}
	finfo, e := file.Stat()
	if e != nil {
		return nil, fmt.Errorf("index.openIdxFile: unexpected: %s", e)
	}
	fsize := uint64(finfo.Size())
	var idx = &idxfile{
		opMode:   opMode,
		file:     file,
		filename: filename,
		size:     fsize,
		poff:     fsize,
	}

	if e := idx.readAndVerifyHeader(); e != nil {
		idx.file.Close()
		return nil, e
	}

	return idx, nil
}

// Writes a new revision of the object.idx file. This function is meaningful
// only for op modes that modify the object index, and have a changeset
// that can be applied.
//
// Returns (true, nil) if openned in IdxUpdate|IdxCompate mode and modified.
// Returns (false, nil) if openned in IdxUpdate|IdxCompate mode but not changed.
// Returns (false, ErrIdxOpMode) if not in a write op mode.
// Returns (false, ErrIdxClosed) if idxfile was closed. REVU this is typical.
func (idx *idxfile) Sync() (bool, error) {
	if idx.opMode == IdxRead {
		return false, ErrIdxOpMode
	}
	if idx.file == nil {
		return false, ErrIdxClosed
	}

	// modset

	// TODO if modset is not nil, then first sort so we
	//      have ascending offsets, and then do the inplace
	//      mod of the record.

	// appendlog

	// TODO if appendlog is not nil seek end and
	//      apply appendlog in sequence. (it is already in order).

	panic("idxfile idxfile.Sync: not implemented")
}

// Closes the object.idx file, regardless of op mode. Further use of the
// idxfile reference will result in panic.
//
// This function will return ErrIdxPendingChanges if called with a non-applied
// changeset (IdxUpdate | IdxCompact modes). To force discard of changes in
// those modes first call Discard().
//
// Otherwise, any returned error is from the underlying os.File.Close.
func (idx *idxfile) Close() error {
	if idx.file == nil {
		return ErrIdxClosed
	}
	if idx.PendingChanges() {
		return ErrIdxPendingChanges
	}

	// if we error out here, then we have a either an os fault or
	// or a bug. Either way, the idxfile object is out of commission.
	e := idx.file.Close()
	idx.file = nil

	return e
}

func (idx *idxfile) PendingChanges() bool {
	return idx.modified
}

// add object - gart-add - oflag must be os.O_RDWR
// append idx record and return offset
func (idx *idxfile) Add(oid *OID, tags, systemics bitmap.Bitmap, date unixtime.Time) (uint64, error) {
	// assert state
	if idx.opMode != IdxUpdate {
		return notIndexed, ErrIdxOpMode
	}

	// create new record
	var header = idxrec_header{
		flags:   idxrec_valid,
		oid:     oid.dat, // REVU we're writing to file so no need to copy
		tbahlen: uint8(len(tags.Bytes())),
		sbahlen: uint8(len(systemics.Bytes())),
	}

	var record = idx_record{
		idxrec_header: header,
		date:          date,
		tags:          tags,
		systemics:     systemics,
	}

	fmt.Printf("debug - record: len:%d\n", record.length())
	fmt.Printf("debug - %s\n", record.String())

	var pending idxPendingOp
	pending.offset = idx.poff
	pending.record = &record

	idx.poff += uint64(record.length())
	idx.appendLog = append(idx.appendLog, pending)

	idx.onUpdate()

	return pending.offset, nil
	/*
		// REVU not here -- changes should be applied at Sync only
		// seek end, write record, get new offset
		roff, e := f.file.Seek(0, os.SEEK_END)
		if e != nil {
			return notIndexed, e
		}

		n, e := record.writeTo(f.file)
		if e != nil {
			return notIndexed, e
		}

		f.roff = uint64(roff + int64(n)) // REVU so do we even need this field? (for read?)
		f.onUpdate()

		// we don't sync
		return uint64(roff), nil
	*/
	panic("idxfile idxfile.Add: not implemented")
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

// Returns the idxfile's source filename.
func (idx *idxfile) Filename() string { return idx.filename }

// Returns the semantic size, i.e. the number of objects
func (idx *idxfile) Size() int { return int(idx.ocnt) }

// Returns a debug string representation. For logging, use idxfile.String()
func (idx *idxfile) DebugStr() string { return fmt.Sprintf("debug: %v", idx) }
