// Doost!

package card

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/index"
)

/// index.Card support types ///////////////////////////////////////////////////

// card file header related consts
const (
	headerBytes    = 32
	card_file_code = 0xec5a011e // sha256("card-file")[:4]
)

// Every card has a fixed width binary header of 64 bytes
type header struct {
	ftype    uint32
	crc32    uint32  // of card file buffer [8:] so no ftype & crc
	created  int64   // unix seconds not nanos
	updated  int64   // unix seconds not nanos
	flags    byte    // 8bits should be fine
	pathcnt  uint8   // 255 instances of an obj should be sufficient
	tbahlen  uint8   // user-tags bah buflen TODO fix read
	sbahlen  uint8   // systemic-tags bah buflen
	reserved [4]byte // TODO 1B for pathcnt -
}

func (p *header) DebugStr() string {
	fp := func(fs string, a ...interface{}) string {
		return fmt.Sprintf(fs, a...)
	}
	var s string
	s += fp("card.header:\n")
	s += fp("\tftype:     %08x\n", p.ftype)
	s += fp("\tcrc32:     %08x\n", p.crc32)
	s += fp("\tcreated-on:%016x\n", p.created)
	s += fp("\tupdated-on:%016x\n", p.updated)
	s += fp("\tflags:     %08b\n", p.flags)
	s += fp("\tpathcnt:   %d\n", p.pathcnt)
	s += fp("\ttbahlen:   %d\n", p.tbahlen)
	s += fp("\tsbahlen:   %d\n", p.sbahlen)
	s += fp("\treserved:  %d\n", p.reserved)
	return s
}

// card type supports the index.Card interface. It has a fixed width header
// and a variable number of associated paths and tags.
// Not all elements of this structure are persisted in the binary image.
type card_t struct {
	header                 // serialized
	oid          index.OID // 32 bytes TODO assert this on init
	tagsBah      []byte    // serialized user tags' bah bitmap - can change
	systemicsBah []byte    // serialized systemic tags' bah bitmap - write once
	paths        []string  // serialized associated fs object paths

	/* not persisted */
	modified bool // REVU on New, add/del tags, add/del paths
}

func (p *card_t) DebugStr() string {
	fp := func(fs string, a ...interface{}) string {
		return fmt.Sprintf(fs, a...)
	}
	var s string = p.header.DebugStr()
	s += fp("------------\n")
	s += fp("\toid:      %x\n", p.oid)
	s += fp("\ttags:     %08b\n", p.tagsBah)
	s += fp("\tsystemics:%08b\n", p.systemicsBah)
	for i, path := range p.paths {
		s += fp("\tpath[%d]: %q\n", i, path)
	}
	s += fp("\tmodified:    %t\n", p.modified)
	s += fp("\tbufsize:  %d\n", p.bufsize())
	return s
}

/// life-cycle ops /////////////////////////////////////////////////////////////

// Card files are created and occasionally modified. the read/write pattern is
// expected to be a quick load, read, and then possibly update and sync.
//
// Like tag.tagmap_t, on updates card_t will first write to a swap file and then
// replace its source file with the updated card data.

// REVU every other func is using fname. the OS FS dirs are a convenient index but
//      should not be considered canonical.
func Exists(oid []byte) bool { panic("card_t.Exists: not implemented") }

// Creates a new card. card_t file is assigned on Card.Save().
func New(oid *index.OID, path string, tagsBah, systemicsBah []byte) (index.Card, error) {
	// accept any value for an oid except all zero bytes.
	if !oid.IsValid() {
		return nil, fmt.Errorf("err - card.New: oid is invalid")
	}
	if len(path) == 0 { // REVU not a library! do not verify path
		return nil, fmt.Errorf("bug - card.New: path is zero-len")
	}
	if len(tagsBah) == 0 {
		return nil, fmt.Errorf("bug - card.New: tagsBah is zero-len")
	}
	if len(systemicsBah) == 0 {
		return nil, fmt.Errorf("bug - card.New: systemicsBah is zero-len")
	}
	// header.crc32 is computed and set at save.
	hdr := header{
		ftype:   card_file_code,
		created: time.Now().Unix(),
		updated: time.Now().Unix(),
		pathcnt: 1,
		tbahlen: uint8(len(tagsBah)),
		sbahlen: uint8(len(systemicsBah)),
	}

	card := &card_t{
		header:       hdr,
		oid:          *oid,
		tagsBah:      tagsBah,
		systemicsBah: systemicsBah,
		paths:        []string{path},
		modified:     true,
	}

	return card, nil
}

// Read an existing card file. File is read in RDONLY mode and immediately closed.
// Use index.Card.Sync() to update the file (if modified).
func Read(fname string) (index.Card, error) {
	finfo, e := os.Stat(fname)
	if e != nil {
		return nil, e
	}
	buf, e := fs.ReadFull(fname)
	if e != nil {
		return nil, e
	}

	hdr, e := readAndVerifyHeader(buf, finfo)
	if e != nil {
		return nil, e
	}

	/// create in-mem card rep ---------------------------------------------------
	var offset = headerBytes
	if offset == len(buf) {
		return nil, fmt.Errorf("bug - card_t.Read: card has no data - fname: %q", fname)
	}

	// read & verify the OID - 32 bytes
	var oid index.OID
	copy(oid[:], buf[offset:offset+len(oid)])
	if !oid.IsValid() {
		return nil, fmt.Errorf("bug - card_t:Read - invalid OID %02x: %d", oid)
	}
	offset += len(oid)

	// read user-tags and systemics-tags BAHs
	tagsBah := buf[offset : offset+int(hdr.tbahlen)]
	offset += int(hdr.tbahlen)

	systemicsBah := buf[offset : offset+int(hdr.sbahlen)]
	offset += int(hdr.sbahlen)

	// read (all) path(s)
	var paths = make([]string, hdr.pathcnt)
	var pathcnt int
	for offset < len(buf) {
		n, path := readLine(buf[offset:])
		paths = append(paths, string(path))
		offset += n
		pathcnt++
	}
	// At least one path must be in the card or we have a bug
	if pathcnt != int(hdr.pathcnt) {
		return nil, fmt.Errorf("bug - card_t.Read: pathcnt exp:%d have:%d - fname: %q",
			hdr.pathcnt, pathcnt, fname)
	}

	return &card_t{
		header:       *hdr,
		oid:          oid,
		tagsBah:      tagsBah,
		systemicsBah: systemicsBah,
		paths:        paths,
	}, nil

	panic("card_t.Read: not implemented")
}

// read until '\n' or end of buffer.
// Return offset at position after the delim.
func readLine(buf []byte) (int, []byte) {
	var xof int //= offset
	for xof < len(buf) {
		if buf[xof] == '\n' {
			break
		}
		xof++
	}
	return xof + 1, buf[:xof]
}

/// interface: index.Card /////////////////////////////////////////////////////
func (c *card_t) RemovePath(fpath string) (bool, error) {
	panic("card_t: index.Card method not implemented")
}

// REVU this 'bah' business is silly. Again, this is not a library!
func (c *card_t) UpdateUserTagBah(bitmap []byte) {
	panic("card_t: index.Card method not implemented")
}

// Save writes the card to a swap file and then rename to file 'fname' as given.
// Save always writes the file, even if card file has not changed. Use Sync in
// conjunction with card.Load(cardfile) if io is to be limited to the case of
// changed cards.
func (c *card_t) Save(fname string) (bool, error) {

	// TODO use the same fs func for tagmap
	swapfile := fs.SwapfileName(fname)
	var abort = true // REVU for now, treat dangling swaps as system bugs
	sfile, existing, e := fs.OpenNewSwapfile(swapfile, abort)
	if e != nil {
		err := fmt.Errorf("bug - card_t.Save: on OpenNewSwapfile - existing:%t - %s", existing, e)
		return false, err
	}
	defer sfile.Close()

	var bufsize = c.bufsize()
	var buf = make([]byte, bufsize)

	if e := c.encode(buf); e != nil {
		return false, e // only bugs
	}

	// write buf to file
	_, e = sfile.Write(buf)
	if e != nil {
		return false, e
	}

	// fsync the swap file
	if e := sfile.Sync(); e != nil {
		return false, fmt.Errorf("bug - card_t.Save: sfile.Sync - %s", e)
	}
	// rename to actual card file
	if e := os.Rename(swapfile, fname); e != nil {
		return false, fmt.Errorf("bug - card_t.Save: os.Rename swp:%q dst:%q - %s",
			swapfile, fname, e)
	}

	panic("card_t: index.Card method not implemented")
}

func (c *card_t) CreatedOn() time.Time   { panic("card_t: index.Card method not implemented") }
func (c *card_t) UpdatedOn() time.Time   { panic("card_t: index.Card method not implemented") }
func (c *card_t) Flags() uint32          { panic("card_t: index.Card method not implemented") }
func (c *card_t) Oid() index.OID         { panic("card_t: index.Card method not implemented") }
func (c *card_t) TagsBitmap() []byte     { panic("card_t: index.Card method not implemented") }
func (c *card_t) SystemicBitmap() []byte { panic("card_t: index.Card method not implemented") }
func (c *card_t) DayTagBah() []byte      { panic("card_t: index.Card method not implemented") }
func (c *card_t) Paths() []string        { panic("card_t: index.Card method not implemented") }
func (c *card_t) AddPath(fpath string) (bool, error) {
	panic("card_t: index.Card method not implemented")
}

func (c *card_t) encode(buf []byte) error {
	var bufsize = c.bufsize()

	if len(buf) < bufsize {
		return fmt.Errorf("bug - card_t.encode: buflen:%d required:%d", len(buf), bufsize)
	}

	*(*uint32)(unsafe.Pointer(&buf[0])) = c.ftype
	*(*int64)(unsafe.Pointer(&buf[8])) = c.created
	*(*int64)(unsafe.Pointer(&buf[16])) = c.updated
	buf[24] = c.flags
	buf[25] = c.pathcnt
	buf[26] = c.tbahlen
	buf[27] = c.sbahlen
	*(*[4]byte)(unsafe.Pointer(&buf[28])) = c.reserved
	*(*index.OID)(unsafe.Pointer(&buf[32])) = c.oid
	var offset = 64
	copy(buf[offset:], c.tagsBah)
	offset += int(c.tbahlen)
	copy(buf[offset:], c.systemicsBah)
	offset += int(c.sbahlen)
	for _, path := range c.paths {
		copy(buf[offset:], []byte(path))
		offset += len(path)
		buf[offset] = '\n'
		offset++
	}

	// finally, compute & encode the checksum
	var crc32 = digest.Checksum32(buf[8:bufsize])
	*(*uint32)(unsafe.Pointer(&buf[4])) = crc32

	// XXX temp asserts
	if offset != bufsize {
		return fmt.Errorf("bug - card_t.encode: offset:%d expected:%d", offset, bufsize)
	}
	// if card wasn't modified then the checksums should be the same
	if !c.modified {
		if c.crc32 != crc32 {
			return fmt.Errorf("bug - card_t.encode: crc32:%08x c.crc32:%08x", crc32, c.crc32)
		}
	}
	// XXX temp assert

	return nil // fini
}

/// internal ops ///////////////////////////////////////////////////////////////

// ? REVU if only a single fs object is associated with this card, return an error.
// TODO index.RemoveObject(oid)

func (c *card_t) bufsize() int {
	n := headerBytes
	n += index.OidBytes
	n += len(c.tagsBah)
	n += len(c.systemicsBah)
	// each path is len of the []byte of path + \n
	for _, p := range c.paths {
		n += len([]byte(p)) + 1
	}
	return n
}

// REVU can we just merge header with card_t ? why not?
func readAndVerifyHeader(buf []byte, finfo os.FileInfo) (*header, error) {
	if len(buf) < headerBytes {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid buffer - len:%d", len(buf))
	}

	var hdr = (*header)(unsafe.Pointer(&buf[0]))

	if hdr.ftype != card_file_code {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid ftype: %04x ", hdr.ftype)
	}

	// TODO correct CRC usage in fix tagmap_t as well.
	var crc32 = digest.Checksum32(buf[4:])
	if hdr.crc32 != crc32 {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid crc32: %04x expect: %04x",
			hdr.crc32, crc32)
	}

	if hdr.created == 0 || hdr.created > hdr.updated {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid created: %d ", hdr.created)
	}
	if hdr.updated == 0 {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid updated: %d ", hdr.updated)
	}

	// reserved must be all 0x00
	for i, v := range hdr.reserved {
		if v != 0x00 {
			return nil, fmt.Errorf("card.readAndVerifyHeader - invalid reserved[%d]: %d", i, v)
		}
	}

	return hdr, nil
}
