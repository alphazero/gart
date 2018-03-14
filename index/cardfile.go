// Doost!

/* cardfile.go: a simple binary blob file per Card implementation */

package index

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/unixtime"
)

/// Card support types ///////////////////////////////////////////////////

// card file header related consts
const (
	headerBytes    = 32
	card_file_code = 0xec5a011e // sha256("card-file")[:4]
)

// Every card has a fixed width binary header of 64 bytes
type header struct {
	ftype    uint32
	crc32    uint32        // of card file buffer [8:] so no ftype & crc
	key      uint64        // 64 bit hash of the OID - likely used as offset
	created  unixtime.Time // unsigned 32bits
	updated  unixtime.Time // unsigned 32bits
	flags    byte          // 8bits should be fine
	pathcnt  uint8         // 255 instances of an obj should be sufficient
	tbahlen  uint8         // user-tags bah buflen TODO fix read
	sbahlen  uint8         // systemic-tags bah buflen
	revision uint16        // new is revision 0
	reserved [2]byte
}

func (p *header) DebugStr() string {
	fp := func(fs string, a ...interface{}) string {
		return fmt.Sprintf(fs, a...)
	}
	var s string
	s += fp("card.header:\n")
	s += fp("\tftype:     %08x\n", p.ftype)
	s += fp("\tcrc32:     %08x\n", p.crc32)
	s += fp("\tkey:     %08x\n", p.key)
	s += fp("\tcreated-on:%08x (%s) \n", p.created, p.created.Date())
	s += fp("\tupdated-on:%08x (%s) \n", p.updated, p.updated.Date())
	s += fp("\tflags:     %08b\n", p.flags)
	s += fp("\tpathcnt:   %d\n", p.pathcnt)
	s += fp("\ttbahlen:   %d\n", p.tbahlen)
	s += fp("\tsbahlen:   %d\n", p.sbahlen)
	s += fp("\trevision:  %d\n", p.revision)
	s += fp("\treserved:  %d\n", p.reserved)
	return s
}

// card type supports the Card interface. It has a fixed width header
// and a variable number of associated paths and tags.
// Not all elements of this structure are persisted in the binary image.
type card_t struct {
	header                  // serialized
	oid       OID           // 32 bytes TODO assert this on init
	tags      bitmap.Bitmap // serialized user tags' bah bitmap - can change
	systemics bitmap.Bitmap // serialized systemic tags' bah bitmap - write once
	paths     []string      // serialized associated fs object paths

	/* not persisted */
	source   string // REVU really belongs to header, only set on LoadOrCreate()
	modified bool   // REVU on New, add/del tags, add/del paths
}

func (p *card_t) DebugStr() string {
	fp := func(fs string, a ...interface{}) string {
		return fmt.Sprintf(fs, a...)
	}
	var s string = p.header.DebugStr()
	s += fp("------------\n")
	s += fp("\toid:      %x\n", p.oid)
	s += fp("\ttags:     %08b\n", p.tags)
	s += fp("\tsystemics:%08b\n", p.systemics)
	for i, path := range p.paths {
		s += fp("\tpath[%d]: %q\n", i, path)
	}
	s += fp("\tsource:   %q\n", p.source)
	s += fp("\tmodified: %t\n", p.modified)
	s += fp("\tbufsize:  %d\n", p.bufsize())
	return s
}

/// life-cycle ops /////////////////////////////////////////////////////////////

// internal use only
func newCard0(oid *OID, key uint64, source string) Card {
	// header.crc32 is computed and set at save.
	hdr := header{
		ftype:   card_file_code,
		created: unixtime.Now(),
		key:     key,
		updated: 0,
		pathcnt: 0,
		tbahlen: 0,
		sbahlen: 0,
	}

	return &card_t{
		header:   hdr,
		oid:      *oid,
		modified: false, // so it can't be saved unless initialized
		source:   source,
	}
}

// Read an existing card file. File is read in RDONLY mode and immediately closed.
// Use Card.Sync() to update the file (if modified).
// Returns (card, nil) on success. Card is nil on errors.
// HERE shouldn't this return index.ErrCardNotFound?
func readCard(garthome string, oid *OID) (Card, error) {

	var cardfile = cardfilePath(garthome, oid)

	finfo, e := os.Stat(cardfile)
	if e != nil && os.IsNotExist(e) {
		return nil, ErrCardNotFound
	} else if e != nil {
		return nil, fmt.Errorf("bug - card_t.readCard: %s", e)
	}

	buf, e := fs.ReadFull(cardfile)
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
		return nil, fmt.Errorf("bug - card_t.Read: card has no data - cardfile: %q", cardfile)
	}

	// read & verify the OID - 32 bytes
	var card_oid OID
	var oidDat = buf[offset : offset+oidBytesLen]
	if e := validateOidBytes(oidDat); e != nil {
		return nil, fmt.Errorf("bug - card_t:Read - %s", e)
	}
	copy(card_oid.dat[:], oidDat)
	if !oid.isEqual(&card_oid) {
		return nil, fmt.Errorf("bug - card_t:Read - cardfile OID - have::%v should-be:%v", card_oid, oid)
	}

	offset += oidBytesLen

	// read user-tags and systemics-tags BAHs
	tagBytes := buf[offset : offset+int(hdr.tbahlen)]
	offset += int(hdr.tbahlen)

	systemicsBytes := buf[offset : offset+int(hdr.sbahlen)]
	offset += int(hdr.sbahlen)

	// read (all) path(s)
	var paths = make([]string, hdr.pathcnt)
	var pathcnt int
	for offset < len(buf) {
		n, path := readLine(buf[offset:])
		paths[pathcnt] = string(path)
		offset += n
		pathcnt++
	}

	if pathcnt != int(hdr.pathcnt) {
		return nil, fmt.Errorf("bug - card_t.Read: pathcnt exp:%d have:%d - cardfile: %q",
			hdr.pathcnt, pathcnt, cardfile)
	}

	return &card_t{
		header:    *hdr,
		oid:       card_oid,
		tags:      bitmap.NewCompressed(tagBytes),
		systemics: bitmap.NewCompressed(systemicsBytes),
		paths:     paths,
		modified:  false,
		source:    cardfile,
	}, nil
}

func (c *card_t) onUpdate() {
	c.updated = unixtime.Now()
	if !c.modified {
		c.modified = true
		c.revision++
	}
}

/// interface: indexedCard ///////////////////////////////////////////////

func (c *card_t) Key() uint64 { return c.key }
func (c *card_t) SetKey(key uint64) {
	if c.key == key {
		return
	}
	c.key = key
	c.onUpdate()
}

/// interface: Card //////////////////////////////////////////////////////

func (c *card_t) UpdateTags(bm bitmap.Bitmap) (bool, error) {
	panic("card_t.UpdateTags: not implemented")
	//	return false, nil
}

func (c *card_t) SetTags(bm bitmap.Bitmap) error {
	bmlen := len(bm.Bytes())
	if bmlen == 0 {
		fmt.Errorf("bug - card_t.UpdateTags: bm is zerolen")
	}
	if bmlen > 255 {
		fmt.Errorf("oops - card_t.UpdateTags: bm is larger than conceived")
	}
	c.tags = bm
	c.tbahlen = uint8(bmlen)

	c.onUpdate()
	//	c.updated = unixtime.Now()
	//	c.modified = true

	return nil
}

func (c *card_t) UpdateSystemics(bm bitmap.Bitmap) (bool, error) {
	panic("card_t.UpdateSystemics: not implemented")
	//	return false, nil
}

func (c *card_t) SetSystemics(bm bitmap.Bitmap) error {
	bmlen := len(bm.Bytes())
	if bmlen == 0 {
		fmt.Errorf("bug - card_t.UpdateTags: bm is zerolen")
	}
	if bmlen > 255 {
		fmt.Errorf("oops - card_t.UpdateTags: bm is larger than conceived")
	}
	c.systemics = bm
	c.sbahlen = uint8(bmlen)

	c.onUpdate()
	//	c.updated = unixtime.Now()
	//	c.modified = true

	return nil
}

// Saves the card. Returns (true, nil) if successful and cardfile was actually written.
// If card was not modified, save is a NOP and returns (false, nil).
//
// panics if Save is called and card_t does not have a 'source' assigned.
func (c *card_t) Save() (bool, error) {

	if c.source == "" {
		panic("bug - card_t.Save: source is zerolen")
	}

	if !c.modified {
		return false, nil
	}

	// TODO use the same fs func for tagmap
	swapfile := fs.SwapfileName(c.source)

	// pass 'true' for abort - for now, treat dangling swaps as system bugs
	sfile, existing, e := fs.OpenNewSwapfile(swapfile, true)
	if e != nil {
		return false, fmt.Errorf("bug - card_t.Save: on OpenNewSwapfile - existing:%t - %s", existing, e)
	}
	defer sfile.Close()

	var bufsize = c.bufsize()
	var buf = make([]byte, bufsize)

	if e := c.encode(buf); e != nil {
		return false, e // can only be a bug
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
	if e := os.Rename(swapfile, c.source); e != nil {
		return false, fmt.Errorf("bug - card_t.Save: os.Rename swp:%q dst:%q - %s",
			swapfile, c.source, e)
	}

	c.modified = false

	return true, nil
}

func (c *card_t) CreatedOn() time.Time    { return c.created.StdTime() }
func (c *card_t) UpdatedOn() time.Time    { return c.updated.StdTime() }
func (c *card_t) Flags() byte             { return c.flags }
func (c *card_t) Oid() OID                { return c.oid }
func (c *card_t) Tags() bitmap.Bitmap     { return c.tags }      // REVU return copy?
func (c *card_t) Systemic() bitmap.Bitmap { return c.systemics } // REVU return copy?
func (c *card_t) Paths() []string         { return c.paths }     // REVU return copy?
func (c *card_t) Revision() int           { return int(c.revision) }

func (c *card_t) AddPath(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("err - card_t.AddPath: invalid arg - zerolen path")
	}
	for _, s := range c.paths {
		if s == path {
			return false, nil // REVU no error on add existing
		}
	}
	c.paths = append(c.paths, path)
	c.pathcnt++
	c.onUpdate()

	return true, nil // REVU return true regardless of onUpdate() effects
}

func (c *card_t) RemovePath(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("err - card_t.RemovePath: invalid arg - zerolen path")
	}

	var i int
	for i < len(c.paths) {
		if path == c.paths[i] {
			if c.pathcnt == 1 {
				return false, fmt.Errorf("err - card_t.RemovePath: illegal state - card's only path")
			}
			goto found
		}
		i++
	}
	return false, nil // not found

found:
	if i != len(c.paths) {
		copy(c.paths[i:], c.paths[i+1:])
	}
	c.pathcnt--
	c.paths = c.paths[:c.pathcnt]
	c.onUpdate()

	return true, nil
}

/// internal ops ///////////////////////////////////////////////////////////////

// REVU crc is being set here -- function name does NOT reflect it.
// TODO rethink this!
func (c *card_t) encode(buf []byte) error {
	var bufsize = c.bufsize()

	if len(buf) < bufsize {
		return fmt.Errorf("bug - card_t.encode: buflen:%d required:%d", len(buf), bufsize)
	}

	// header's fields
	*(*uint32)(unsafe.Pointer(&buf[0])) = c.ftype
	*(*uint64)(unsafe.Pointer(&buf[8])) = c.key
	*(*uint32)(unsafe.Pointer(&buf[16])) = c.created.Timestamp()
	*(*uint32)(unsafe.Pointer(&buf[20])) = c.updated.Timestamp()
	buf[24] = c.flags
	buf[25] = c.pathcnt
	buf[26] = c.tbahlen
	buf[27] = c.sbahlen
	*(*uint16)(unsafe.Pointer(&buf[28])) = c.revision
	*(*[2]byte)(unsafe.Pointer(&buf[30])) = c.reserved

	// HERE REVU should be part of header ?
	*(*OID)(unsafe.Pointer(&buf[32])) = c.oid

	// card_t's persisted fields
	var offset = headerBytes + oidBytesLen //64
	copy(buf[offset:], c.tags.Bytes())
	offset += int(c.tbahlen)
	copy(buf[offset:], c.systemics.Bytes())
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
	// also update the card's crc
	c.crc32 = crc32

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

// REVU can we just merge header with card_t ? why not?
func readAndVerifyHeader(buf []byte, finfo os.FileInfo) (*header, error) {
	if len(buf) < headerBytes {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid buffer - len:%d", len(buf))
	}

	var hdr = (*header)(unsafe.Pointer(&buf[0]))

	if hdr.ftype != card_file_code {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid ftype: %04x ", hdr.ftype)
	}

	//	// check filesize:
	//	if hdr.bufsize != finfo.Size() {
	//		return nil, fmt.Error("bug - card.readAndVerifyHeader - hdr.bufsize:%d finfo.Size:%d",
	//			hdr.bufsize, finfo.Size())
	//	}

	// TODO correct CRC usage in fix tagmap_t as well.
	var crc32 = digest.Checksum32(buf[8:])
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

	// At least one path must be in the card or we have a bug
	if hdr.pathcnt == 0 {
		return nil, fmt.Errorf("card.readAndVerifyHeader - invalid pathcnt: %d", hdr.pathcnt)
	}

	// reserved must be all 0x00
	for i, v := range hdr.reserved {
		if v != 0x00 {
			return nil, fmt.Errorf("card.readAndVerifyHeader - invalid reserved[%d]: %d", i, v)
		}
	}

	return hdr, nil
}

func (c *card_t) bufsize() int {
	n := headerBytes
	n += oidBytesLen
	n += len(c.tags.Bytes())
	n += len(c.systemics.Bytes())
	// each path is len of the []byte of path + \n
	for _, p := range c.paths {
		n += len([]byte(p)) + 1
	}
	return n
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
