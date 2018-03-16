// Doost!

package index

import (
	"fmt"
	"io"
	"os"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/unixtime"
)

var _ = fs.OpenNewFile

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	idx_file_code = 0x763f079cf73c668e // sha256("index-file")[:8]
)

/// object.idx file ////////////////////////////////////////////////////////////

type idxfile_header struct {
	ftype    uint64
	crc64    uint64        // REVU this is not practical TODO need better solution
	created  unixtime.Time // unsigned 32bits
	updated  unixtime.Time // unsigned 32bits
	revision uint64
	reserved [4080]byte // reserved XXX fix size
}

type idxOp int

const (
	IdxUpdate = idxOp(os.O_RDWR)
	IdxRead   = idxOp(os.O_RDONLY)
)

type idxfile struct {
	idxfile_header
	opflag   idxOp
	file     *os.File
	filename string
	offset   uint64
	modified bool
}

// index.dat file record header
// fixed length portion
type idxrec_header struct {
	flags   byte              // deleted, updated, what else?
	oid     [oidBytesLen]byte // ref. to index.idx.file record
	tbahlen uint8
	sbahlen uint8
}

// index.dat file record
// complete record (fixed header, varlength data)
type idx_record struct {
	header    idxrec_header
	tags      bitmap.Bitmap
	systemics bitmap.Bitmap
	date      unixtime.Time
}

/// ops ////////////////////////////////////////////////////////////////////////

var (
	ErrIdxOpMode = fmt.Errorf("object.idx: illegal state - idxOpMode")
)

// init - used by gart-init
// create the minimal object.idx file:
// idxfile_header and buflen
func createIdxFile(garthome string) error {
	panic("idxfile createIdxFile: not implemented")
}

// open
//	gart-add, gart-tag:     os.O_RDWR   flag
//  gart-find, [gart-list]: os.O_RDONLY flag
func openIdxFile(garthome string, op idxOp) (*idxfile, error) {
	panic("idxfile openIdxFile: not implemented")
}

// add object - gart-add - oflag must be os.O_RDWR
// append idx record and return offset
func (f *idxfile) Add(oid *OID, tags, systemics bitmap.Bitmap, date unixtime.Time) (uint64, error) {
	// assert state
	if f.opflag != IdxUpdate {
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
