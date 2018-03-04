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

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	headerSize       = 4096
	tagmap_file_type = 0xd42b72e897893438 // sha256("tagmap-file")
)

/// types /////////////////////////////////////////////////////////////////////

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
	Sync() (flushed bool, e error)
}

// fs block size (4k) header for the tagmap file
type header struct {
	ftype    uint64     // filetype invariant
	created  int64      // unix nano
	updated  int64      // unix nano
	crc64    uint64     // map data crc
	tagcnt   uint64     // number of Tag entries
	buflen   int64      // file size, // REVU isn't it excluding header ?
	reserved [4044]byte // reserved
}

type tagref struct {
	offset int
	blen   int
}

// tagmap in-mem model
type tagmap struct {
	header
	source    string
	buf       []byte
	changeset []Tag
	m         map[string]tagref
}

func init() {
	// be pedantic and verify all size assumptions
	// header size
	var hdr header
	if unsafe.Sizeof(hdr) != headerSize {
		panic("bug - tagmap header struct size non-conformant")
	}
}

// Size returns the number of tags
func (t *tagmap) Size() uint64 { return t.header.tagcnt }

func (t *tagmap) CreatedOn() int64 { return t.header.created }
func (t *tagmap) UpdatedOn() int64 { return t.header.updated }

// TODO correct the comment
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

	finfo, e := os.Stat(fname)
	if e != nil {
		return nil, e
	}
	buf, e := fs.ReadFull(fname)
	if e != nil {
		return nil, e
	}

	// header.tagcnt still needs to be verified (see below)
	hdr, e := readAndVerifyHeader(buf, finfo)
	if e != nil {
		return nil, e
	}

	/// create in-mem tagmap rep /////////////////////////////////////////////////

	//	buf = buf[headerSize:] // trim header bytes

	// build the in-mem tagmap
	var mapsize = int(float64(hdr.tagcnt) * 1.25) // a bit larger to prevent resize
	var m = make(map[string]tagref, mapsize)
	var id, offset = 0, headerSize
	for offset < len(buf) {
		id++
		if id&0x7 == 0 { // skip multiples of 8 so we don't have to encode7 for BAH
			id++
		}
		var tag Tag // REVU do we need Tag?
		tlen, e := tag.decode(buf[offset:])
		if e != nil {
			return nil, fmt.Errorf("bug - decoding tag[id:%d] offset:%d - %s", id, offset, e)
		}
		m[tag.name] = tagref{
			offset: offset,
			blen:   tlen,
		}
		offset += tlen
	}
	// verify hdr.tagcnt
	tagcnt := uint64(len(m))
	if hdr.tagcnt != tagcnt {
		return nil, fmt.Errorf("LoadTagmap - header - invalid tagcnt: %d loaded: %d ",
			hdr.tagcnt, tagcnt)
	}

	var tmap = &tagmap{
		source: fname,
		header: *hdr,
		m:      m,
		buf:    buf[headerSize:],
	}

	// build mapping to offsets
	return tmap, nil
}

func readAndVerifyHeader(buf []byte, finfo os.FileInfo) (*header, error) {
	if len(buf) < headerSize {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid buffer - len:%d", len(buf))
	}

	var hdr = (*header)(unsafe.Pointer(&buf[0]))

	if hdr.ftype != tagmap_file_type {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid ftype: %04x ", hdr.ftype)
	}
	//	if hdr.flags != 0x00 {
	//		return nil, fmt.Errorf("readAndVerifyHeader - invalid flags: %04x", hdr.flags)
	//	}
	if hdr.created == 0 || hdr.created > hdr.updated {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid created: %d ", hdr.created)
	}
	if hdr.updated == 0 {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid updated: %d ", hdr.updated)
	}
	var crc64 = digest.Checksum64(buf[headerSize:])
	if hdr.crc64 != crc64 {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid crc64: %08x expect: %08x",
			hdr.crc64, crc64)
	}
	if hdr.buflen != finfo.Size()-headerSize {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid buflen: %d expect: %d",
			hdr.buflen, finfo.Size()-headerSize)
	}
	for i, v := range hdr.reserved {
		if v != 0x00 {
			return nil, fmt.Errorf("readAndVerifyHeader - invalid reserved[%d]: %d", i, v)
		}
	}
	return hdr, nil
}

// Sync changes (if any) to source file.
// If tagmap has not changed since loading returns (false, nil)
// Successful sync of a modified tagmap will return (true, nil)
// If error is not nil, attempt to sync is implicit (_, some-error)
func (t *tagmap) Sync() (bool, error) {

	if t.changeset == nil {
		return false, nil
	}

	swapfile := fs.SwapfileName(t.source)
	var ops = os.O_WRONLY | os.O_APPEND
	sfile, e := fs.OpenNewFile(swapfile, ops)
	if e != nil {
		return false, e
	}
	defer sfile.Close()

	// update relevant header bits. buflen should already be correct.
	t.header.updated = time.Now().UnixNano()
	t.header.crc64 = digest.Checksum64(t.buf)

	hdrbuf := *(*[headerSize]byte)(unsafe.Pointer(&t.header))
	_, e = sfile.Write(hdrbuf[:])
	if e != nil {
		return false, e
	}
	_, e = sfile.Write(t.buf[:])
	if e != nil {
		return false, e
	}

	// delete the original file.
	// rename the swap file
	// NOTE syscall.Exchangedata will atomically swap the files.
	//      but may not be supported by all FSs.

	// TODO flush and close the swap file.

	panic(" - not implemented")
}

// creates a new gart tagmap file. This only writes the header.
// File is closed on return.
func createTagmapFile(fname string) error {
	var ops = os.O_WRONLY | os.O_APPEND
	file, e := fs.OpenNewFile(fname, ops)
	if e != nil {
		return e
	}
	defer file.Close()

	// the initial header.
	var now = time.Now().UnixNano()
	var hdr header
	hdr.ftype = tagmap_file_type
	//	hdr.flags = 0x00
	hdr.created = now
	hdr.updated = now
	hdr.tagcnt = 0
	hdr.buflen = 0
	hdr.crc64 = digest.Checksum64([]byte{})

	var arr = *(*[headerSize]byte)(unsafe.Pointer(&hdr))
	_, e = file.Write(arr[:])
	if e != nil {
		return e
	}

	return file.Sync()
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
