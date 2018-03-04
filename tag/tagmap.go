// Doost!

package tag

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
)

/// types /////////////////////////////////////////////////////////////////////

// tagmpa errors
var (
	E_Closed    = fmt.Errorf("tagmap is closed")
	E_NotExists = fmt.Errorf("tagmap file does not exist")
	E_Exists    = fmt.Errorf("tagmap file already exists")
)

// tagmap in-mem model
type tagmap struct {
	header
	buf []byte // serialized tag list binary data from persistent image
	m   map[string]tagref
}

type tagref struct {
	offset int
	blen   int
}

const headerSize = 4096

// fs block size (4k) header for the tagmap file
type header struct {
	size     uint64     // number of Tag entries
	created  int64      // unix nano
	updated  int64      // unix nano
	crc64    uint64     // map data crc
	reserved [4064]byte // reserved
}

// be pedantic and verify all size assumptions
func init() {
	// header size
	var hdr header
	if unsafe.Sizeof(hdr) != headerSize {
		panic("bug - tagmap header struct size non-conformant")
	}
}

// using an interface to facilitate swapping the implementation (thinking B-Tree).
type Tagmap interface {
	Size() uint64
	CreatedOn() int64
	UpdatedOn() int64

	// Adds a new tag.
	// added is true if tag was indeed new. Otherwise false (with no error)
	// Error is returned if tag name exceeds maximum tag name byteslen.
	Add(string) (added bool, err error)
	// Increments the named tag's refcnt and returns the new refcnt.
	// returns error if tag does not exist.
	IncrRefcnt(string) (refcnt int, err error)
	// returns ids of selected tags.
	// notDefined is never nil. If not empty, it contains all
	// tag names that are not defined.
	SelectTags([]string) (ids []int, notDefined []string)
	// Syncs the tagmap file. IFF the in-mem model has been modified
	// changes are flushed to the disk. This is a blocking call.
	//
	// if flushed is true, n > 0 (if e == nil) and is the size of the
	// file. If flushed is false, n ==0, e == nil.
	// If flushed & error != nil, n (may) reflect how many bytes were
	// actually written.
	Sync() (flushed bool, n int, e error)
}

func (t *tagmap) Size() uint64     { return t.header.size }
func (t *tagmap) CreatedOn() int64 { return t.header.created }
func (t *tagmap) UpdatedOn() int64 { return t.header.updated }

func (t *tagmap) Sync() (bool, int, error) {
	panic(" - not implemented")
}

func (t *tagmap) Add(tag string) (bool, error) {
	panic(" - not implemented")
}

func (t *tagmap) SelectTags([]string) (ids []int, notDefined []string) {
	panic(" - not implemented")
}

func (t *tagmap) IncrRefcnt(tag string) (int, error) {
	panic(" - not implemented")
}

// Load loads tagmap from the named file. If file does not
// exist and create is true, it will create a zero-entry tagmap
// file. If create is false and file does not exist, E_NotExists
// error is returned. Other file IO related errors are propagated
// as encountered.
//
// On successful Open, a Tagmap is returned. If error is not nil,
// the Tagmap result is nil
//
// NOTE this is not a library but these functions need to be exported. That said
// it is the assumption that these functions are called by gart processes and we need
// not worry about the argument's validity or other related matter.
func LoadTagmap(fname string, create bool) (Tagmap, error) {

	if create {
		// reminder that this will close the new tagmap file on success
		if e := createTagmapFile(fname); e != nil {
			return nil, e
		}
	}

	// read the tagmap file
	buf, e := fs.ReadFull(fname)
	if e != nil {
		return nil, e
	}

	// create in-mem rep.

	// read header for tagmap size, etc.
	var hdr = *(*header)(unsafe.Pointer(&buf[0]))
	var mapsize = int(float64(hdr.size) * 1.25) // a bit larger to prevent resize

	// build the in-mem tagmap
	var tmap = &tagmap{
		header: hdr,
		m:      make(map[string]tagref, mapsize),
		buf:    buf[headerSize:],
	}

	// build mapping to offsets
	// we assign ids in order. For all ids, id mod 8 != 0 to facilitate building
	// the encode7 groups for BAH compression.
	var id int = 1
	var offset int
	for offset < len(tmap.buf) {
		var tag Tag
		tlen, e := tag.decode(buf[offset:])
		if e != nil {
			return nil, fmt.Errorf("bug - decoding tag[id:%d] offset:%d - %s", id, offset, e)
		}
		tmap.m[tag.name] = tagref{
			offset: offset,
			blen:   tlen,
		}
		id++
		if id&0x8 == 0 {
			id++
		}
	}

	return tmap, nil
}

// creates a new gart tagmap file. This only writes the header.
// File is closed on return.
func createTagmapFile(fname string) error {
	var flags = os.O_CREATE | os.O_EXCL | os.O_WRONLY | os.O_APPEND | os.O_SYNC
	file, e := os.OpenFile(fname, flags, fs.FilePerm)
	if e != nil {
		return e
	}
	defer file.Close()

	// the initial header.
	var now = time.Now().UnixNano()
	var hdr header
	hdr.created = now
	hdr.updated = now
	hdr.crc64 = digest.Checksum64([]byte{})

	var arr = *(*[headerSize]byte)(unsafe.Pointer(&hdr))
	_, e = file.Write(arr[:])
	if e != nil {
		return e
	}

	return file.Sync()
}
