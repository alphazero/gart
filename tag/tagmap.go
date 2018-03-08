// Doost!

/* tagmap.go: varlen record flatfile, map[] in-mem implementation of tag.Map */

package tag

import (
	"fmt"
	"os"
	"sort"
	"time"
	"unsafe"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
)

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	headerBytes      = 4096               // minimum length of a tagmap.dat file
	tagmap_file_code = 0xd42b72e897893438 // sha256("tagmap-file")[:8]
)

/// types /////////////////////////////////////////////////////////////////////

// fs block size (4k) header for the tagmap file
type header struct {
	ftype    uint64     // filetype invariant
	created  int64      // unix nano
	updated  int64      // unix nano
	crc64    uint64     // map data crc
	tagcnt   uint64     // number of Tag entries
	buflen   uint64     // tag data bytes
	reserved [4044]byte // reserved
}

// tagmap in-mem model
type tagmap struct {
	header          // serialized
	buf      []byte // serlialized
	source   string
	modified bool
	nextId   int
	m        map[string]*Tag
}

func init() {
	// be pedantic and verify all size assumptions

	// header size
	var hdr header
	if unsafe.Sizeof(hdr) != headerBytes {
		panic("bug - tagmap header struct size non-conformant")
	}
}

/// interface: tag.Map ////////////////////////////////////////////////////////

// Size returns the number of tags
func (t *tagmap) Size() uint64         { return t.header.tagcnt }
func (t *tagmap) CreatedOn() time.Time { return time.Unix(t.header.created, 0) }
func (t *tagmap) UpdatedOn() time.Time { return time.Unix(t.header.updated, 0) }

// Load loads tagmap from the named file. If file does not
// exist and create is true, it will create a zero-entry tagmap
// file. If create is false and file does not exist, E_NotExists
// error is returned. Other file IO related errors are propagated
// as encountered.
//
// On successful Open, a Map is returned. If error is not nil,
// the Map result is nil
//
// NOTE this is not a library but these functions need to be exported. That said
// it is the assumption that these functions are called by gart processes and we need
// not worry about the argument's validity or other related matter.
func LoadMap(fname string, create bool) (Map, error) {

	if create {
		// reminder that this will close the new tagmap file on success
		if e := createMapFile(fname); e != nil {
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

	/// create in-mem tagmap rep -------------------------------------------------

	// build the in-mem tagmap
	var mapsize = int(float64(hdr.tagcnt) * 1.25) // a bit larger to prevent resize
	var m = make(map[string]*Tag, mapsize)
	var id int
	var offset = uint64(headerBytes)
	for offset < uint64(len(buf)) {
		id++
		if id&0x7 == 0 { // skip multiples of 8 so we don't have to encode7 for BAH
			id++
		}
		var tag Tag // REVU do we need Tag?
		tlen, e := tag.decode(buf[offset:])
		if e != nil {
			return nil, fmt.Errorf("bug - decoding tag[id:%d] offset:%d - %s", id, offset, e)
		}
		tag.id = id
		tag.offset = offset - headerBytes
		m[tag.name] = &tag

		offset += uint64(tlen)
	}
	// verify hdr.tagcnt
	tagcnt := uint64(len(m))
	if hdr.tagcnt != tagcnt {
		return nil, fmt.Errorf("LoadMap - header - invalid tagcnt: %d loaded: %d ",
			hdr.tagcnt, tagcnt)
	}

	var tmap = &tagmap{
		source: fname,
		header: *hdr,
		m:      m,
		buf:    buf[headerBytes:],
		nextId: id + 1,
	}

	// build mapping to offsets
	return tmap, nil
}

// Sync changes (if any) to source file.
// If tagmap has not changed since loading returns (false, nil)
// Successful sync of a modified tagmap will return (true, nil)
// If error is not nil, attempt to sync is implicit (_, some-error)
func (t *tagmap) Sync() (bool, error) {

	if !t.modified {
		return false, nil
	}

	swapfile := fs.SwapfileName(t.source)
	var ops = os.O_WRONLY | os.O_APPEND
	sfile, e := fs.OpenNewFile(swapfile, ops)
	if e != nil {
		return false, e
	}
	defer sfile.Close() // REVU not sure about behavior of defering this due to os.Rename below.

	hdrbuf := *(*[headerBytes]byte)(unsafe.Pointer(&t.header))
	_, e = sfile.Write(hdrbuf[:])
	if e != nil {
		return false, e
	}
	_, e = sfile.Write(t.buf)
	if e != nil {
		return false, e
	}

	if e := os.Rename(swapfile, t.source); e != nil {
		return false, fmt.Errorf("tagmap.Sync - os.Replace %q %q - err: %s", e)
	}

	return true, nil
}

// REVU gart is a tool and gart/tag is NOT a library function.
//      illegal arguments or invalid state errors are bugs.
func (t *tagmap) Add(tagname string) (bool, error) {

	// eat the cost of this minor struct alloc since all constraint
	// checking and name normalization is already done in newTag.
	// Offset is buflen as tag will be appended to t.buf
	tag, e := newTag(tagname, t.nextId, t.header.buflen)
	if e != nil {
		return false, fmt.Errorf("tagmap.Add: invalid argument - name %q", tagname)
	}
	if _, found := t.m[tag.name]; found {
		return false, nil
	}

	// add tag
	t.m[tag.name] = tag
	t.buf = append(t.buf, make([]byte, tag.buflen())...)
	n, e := tag.encode(t.buf[tag.offset:])
	if e != nil {
		panic(fmt.Errorf("bug - tagmap.Add: unexpected error - %s", e))
	}

	t.header.buflen += uint64(n)
	t.header.tagcnt++
	t.nextId++

	t.onUpdate()
	return true, nil
}

func (t *tagmap) IncrRefcnt(tagname string) (int, error) {
	name, ok := normalizeName(tagname)
	if !ok {
		return 0, fmt.Errorf("tagmap.IncrRefcnt: invalid argument - name %q", tagname)
	}
	tag, found := t.m[name]
	if !found {
		return 0, fmt.Errorf("tagmap.IncrRefcnt: no such tag - name %q", tagname)
	}

	// update tag
	tag.refcnt++
	_, e := tag.encode(t.buf[tag.offset:]) // REVU this rewrites the entire tag ..
	if e != nil {
		panic(fmt.Errorf("bug - tagmap.IncrRefcnt: unexpected error - %s", e))
	}

	t.onUpdate()
	return int(tag.refcnt), nil
}

// NOTE output ids should be sorted ascending 1, 2, 7, ..., n
func (t *tagmap) SelectTags(tags []string) (ids []int, notDefined []string) {
	for _, s := range tags {
		if name, ok := normalizeName(s); ok {
			if tag, found := t.m[name]; found {
				ids = append(ids, tag.id)
			} else {
				notDefined = append(notDefined, s)
			}
		} else { // s invalid REVU panic? {
			notDefined = append(notDefined, s)
		}
	}

	// sort ids ascending
	sort.IntSlice(ids).Sort()

	return
}

func (t *tagmap) Tags() []Tag {
	var tags = make([]Tag, t.header.tagcnt)
	var i int
	for _, t := range t.m {
		tags[i] = *t
		i++
	}
	return tags
}

/// internal ops //////////////////////////////////////////////////////////////

// creates a new gart tagmap file. This only writes the header.
// File is closed on return.
func createMapFile(fname string) error {
	var ops = os.O_WRONLY | os.O_APPEND
	file, e := fs.OpenNewFile(fname, ops)
	if e != nil {
		return e
	}
	defer file.Close()

	// the initial header.
	var now = time.Now().Unix()
	var hdr header
	hdr.ftype = tagmap_file_code
	//	hdr.flags = 0x00
	hdr.created = now
	hdr.updated = now
	hdr.tagcnt = 0
	hdr.buflen = 0
	hdr.crc64 = digest.Checksum64([]byte{})

	var arr = *(*[headerBytes]byte)(unsafe.Pointer(&hdr))
	_, e = file.Write(arr[:])
	if e != nil {
		return e
	}

	return file.Sync()
}

func readAndVerifyHeader(buf []byte, finfo os.FileInfo) (*header, error) {
	if len(buf) < headerBytes {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid buffer - len:%d", len(buf))
	}

	var hdr = (*header)(unsafe.Pointer(&buf[0]))

	if hdr.ftype != tagmap_file_code {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid ftype: %04x ", hdr.ftype)
	}
	if hdr.created == 0 || hdr.created > hdr.updated {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid created: %d ", hdr.created)
	}
	if hdr.updated == 0 {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid updated: %d ", hdr.updated)
	}
	var crc64 = digest.Checksum64(buf[headerBytes:])
	if hdr.crc64 != crc64 {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid crc64: %08x expect: %08x",
			hdr.crc64, crc64)
	}
	if hdr.buflen != uint64(finfo.Size())-headerBytes {
		return nil, fmt.Errorf("readAndVerifyHeader - invalid buflen: %d expect: %d",
			hdr.buflen, finfo.Size()-headerBytes)
	}
	for i, v := range hdr.reserved {
		if v != 0x00 {
			return nil, fmt.Errorf("readAndVerifyHeader - invalid reserved[%d]: %d", i, v)
		}
	}

	return hdr, nil
}

// updates tagmap checksum, timestamp, and dirty flag
func (t *tagmap) onUpdate() {
	t.header.crc64 = digest.Checksum64(t.buf)
	t.header.updated = time.Now().Unix()
	t.modified = true
}

/// debug /////////////////////////////////////////////////////////////////////

// TODO do both Debug() and String()

// digest info suitable for log/debug
func (h header) String() string {
	return fmt.Sprintf("ctime:%d utime:%d crc:%016x buflen:%d tagcnt:%d",
		h.created, h.updated, h.crc64, h.buflen, h.tagcnt)
}

// digest info suitable for log/debug
func (t tagmap) String() string {
	return fmt.Sprintf("%s next-id:%d", t.header, t.nextId)
}
