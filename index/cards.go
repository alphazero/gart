// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
)

// An index.Card describes, in full, a gart object. Cards are stored as individual
// files in a .gart/index/cards/ child directory using oids for nameing the dir
// hierarchy.

// REVU - we can load a card but short of searching every tagmap for card.key
//      there is no O(1) way to also show what tags have been applied to the object.
//      - we can add back the 1.0/BAH for tags and the update tags() for Card, but
//      there are issues: (1) BAH would need to be cleaned up to the standard of WAHL
//      and (2) the simple (and sufficient) approach of simply writing the csv tag-line
//      is far simpler and unlike 1.0/card/bah there is no limit on number of tags.

type Card interface {
	Oid() *system.Oid
	Key() int64
	Type() system.Otype // REVU use for systemics tags ..
	Version() int
	Print(io.Writer)
	Tags() []string

	setKey(int64) error               // index use only
	addTag(tag ...string) []string    // returns updated tags, if any
	removeTag(tag ...string) []string // returns removed tags, if any
	isModified() bool
	save() (bool, error) // index use only
}

const cardHeaderSize = 40

type cardFileHeader struct {
	crc32   uint32
	otype   system.Otype
	version int16
	flags   byte
	created int64
	updated int64
	key     int64
	tagcnt  int32
	tagslen int32
}

// len: 4       1       2         1        8         8         8
// xof: 0       4       5         7        8         16        24
// fld: crc32 / otype / version / flags /  created / updated / key
// NOTE encode is written with mmap in mind. It is assumed that the buffer
// is the full card file and the data already mapped.
func (h *cardFileHeader) encode(buf []byte) error {
	if len(buf) < cardHeaderSize {
		return errors.Bug("Invalid arg - buf len:%d < %d", len(buf), cardHeaderSize)
	}
	*(*byte)(unsafe.Pointer(&buf[4])) = byte(h.otype)
	*(*int16)(unsafe.Pointer(&buf[5])) = h.version
	*(*byte)(unsafe.Pointer(&buf[7])) = h.flags
	*(*int64)(unsafe.Pointer(&buf[8])) = h.created
	*(*int64)(unsafe.Pointer(&buf[16])) = h.updated
	*(*int64)(unsafe.Pointer(&buf[24])) = h.key
	*(*int32)(unsafe.Pointer(&buf[32])) = h.tagcnt
	*(*int32)(unsafe.Pointer(&buf[36])) = h.tagslen

	h.crc32 = digest.Checksum32(buf[4:])
	*(*uint32)(unsafe.Pointer(&buf[0])) = h.crc32

	return nil
}

// NOTE decode is written with mmap in mind. It is assumed that the buffer
// is the full card file and the data already mapped.
func (h *cardFileHeader) decode(buf []byte) error {
	h.crc32 = *(*uint32)(unsafe.Pointer(&buf[0]))
	crc32 := digest.Checksum32(buf[4:])
	if crc32 != h.crc32 {
		return errors.Bug("cardFileHeader.decode: computed crc %d != recorded crc:%d", crc32, h.crc32)
	}
	h.otype = system.Otype(*(*byte)(unsafe.Pointer(&buf[4])))
	h.version = *(*int16)(unsafe.Pointer(&buf[5]))
	h.flags = *(*byte)(unsafe.Pointer(&buf[7]))
	h.created = *(*int64)(unsafe.Pointer(&buf[8]))
	h.updated = *(*int64)(unsafe.Pointer(&buf[16]))
	h.key = *(*int64)(unsafe.Pointer(&buf[24]))
	h.tagcnt = *(*int32)(unsafe.Pointer(&buf[32]))
	h.tagslen = *(*int32)(unsafe.Pointer(&buf[36]))
	return nil
}

type cardFile struct {
	// pseudo header
	header   *cardFileHeader
	tags     map[string]struct{}
	datalen  int64 // REVU is this even necessary?
	oid      *system.Oid
	source   string
	buf      []byte // from mmap
	modified bool
	encode   func([]byte) error
}

func (h *cardFileHeader) Print(w io.Writer) {
	fmt.Fprintf(w, "crc32:     %08x\n", h.crc32)
	fmt.Fprintf(w, "type:      %s\n", h.otype)
	fmt.Fprintf(w, "version:   %d\n", h.version)
	fmt.Fprintf(w, "flags:     %08b\n", h.flags)
	fmt.Fprintf(w, "created:   %d - %s\n", h.created, time.Unix(0, h.created))
	fmt.Fprintf(w, "updated:   %d - %s\n", h.updated, time.Unix(0, h.updated))
	fmt.Fprintf(w, "key:       %d\n", h.key)
	fmt.Fprintf(w, "tagcnt:    %d\n", h.tagcnt)
	fmt.Fprintf(w, "tagslen:   %d\n", h.tagslen)
}

func (c *cardFile) Print(w io.Writer) {
	fmt.Fprintf(w, "--- card ---------------\n")
	c.header.Print(w)
	fmt.Fprintf(w, "oid:       %s\n", c.oid.Fingerprint())
	fmt.Fprintf(w, "source:    %q\n", cardFilename(c.oid)) // XXX c.source)
	fmt.Fprintf(w, "data-len:  %d\n", c.datalen)
	fmt.Fprintf(w, "modified:  %t\n", c.modified)
	if len(c.tags) > 0 {
		fmt.Fprintf(w, "tags:        \n")
		tags := c.Tags() // this sorts them
		for n, tag := range tags {
			fmt.Fprintf(w, "\t [%d]:   %q\n", n, tag)
		}
	}
}

/// cardFile ///////////////////////////////////////////////////////////////////

func cardFilename(oid *system.Oid) string {
	oidstr := oid.String()
	return filepath.Join(system.IndexCardsPath, oidstr[:2], oidstr[2:])
}

func cardExists(oid *system.Oid) bool {
	var filename = cardFilename(oid)
	if _, e := os.Stat(filename); e != nil && os.IsNotExist(e) {
		return false
	} else if e != nil {
		panic(errors.Bug("index.cardExists: file: %s - %x", filename, e))
	}
	return true
}

func newCardFile(oid *system.Oid, otype system.Otype) (*cardFile, error) {
	if e := otype.Verify(); e != nil {
		return nil, e
	}
	// REVU we sh/could check if card exists here ..

	now := time.Now().UnixNano()
	header := &cardFileHeader{
		otype:   otype,
		version: -1,
		flags:   0,
		crc32:   0,
		created: now,
		updated: 0,
		key:     -1, // REVU change these -1s to index.invalidKey (objects.go)
	}

	card := &cardFile{
		header:   header,
		tags:     make(map[string]struct{}, 0),
		oid:      oid,
		modified: false,
	}

	return card, nil
}

/// Card support ///////////////////////////////////////////////////////////////

func (c *cardFile) Oid() *system.Oid   { return c.oid }
func (c *cardFile) Key() int64         { return c.header.key }
func (c *cardFile) Type() system.Otype { return c.header.otype }
func (c *cardFile) Version() int       { return int(c.header.version) }
func (c *cardFile) setKey(key int64) error {
	if c.header.key != -1 {
		return errors.Bug("cardFile.setKey: key is already set to %d", c.header.key)
	}
	if key < 0 {
		return errors.InvalidArg("cardFile.setKey", "key", "< 0")
	}

	c.header.key = key
	c.onUpdate()
	return nil
}
func (c *cardFile) isModified() bool { return c.modified }

func (c *cardFile) Tags() []string {
	var tags = make([]string, len(c.tags))
	var n int
	for tag, _ := range c.tags {
		tags[n] = tag
		n++
	}
	sort.Strings(tags)
	return tags
}

func (c *cardFile) addTag(tags ...string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	var updates []string
	for _, tag := range tags {
		if _, ok := c.tags[tag]; !ok {
			c.tags[tag] = struct{}{}
			c.header.tagcnt++
			c.header.tagslen += int32(len(tag) + 1) // 1 for the ,
			updates = append(updates, tag)
		}
	}
	if len(updates) > 0 {
		//		c.header.tagslen--
		c.onUpdate()
	}
	return updates
}

func (c *cardFile) removeTag(tags ...string) []string {
	if len(tags) == 0 || len(c.tags) == 0 {
		return []string{}
	}
	var updates []string
	for _, tag := range tags {
		if _, ok := c.tags[tag]; ok {
			delete(c.tags, tag)
			c.header.tagcnt--
			c.header.tagslen -= int32(len(tag) + 1) // 1 for the ,
			updates = append(updates, tag)
		}
	}
	if len(updates) > 0 {
		if c.header.tagslen < 0 { // edge case of removing the only tag
			c.header.tagslen = 0
		}
		c.onUpdate()
	}
	return updates
}

func (c *cardFile) onUpdate() {
	if !c.modified {
		c.modified = true
		c.header.version++
		c.header.updated = time.Now().UnixNano()
	}
}

func (c *cardFile) debugStr() string {
	return fmt.Sprintf("%s-card:(oid:%s key:%d)",
		c.header.otype, c.oid.Fingerprint(), c.header.key)
}

/// io ops /////////////////////////////////////////////////////////////////////

// REVU this would have to partially create cardFile and then pass it to newTypeCard
func LoadCard(oid *system.Oid) (Card, error) {
	if oid == nil {
		return nil, errors.InvalidArg("index.LoadCard", "oid", "nil")
	}

	/// open file and map it ////////////////////////////////////////

	filename := cardFilename(oid)
	finfo, e := os.Stat(filename)
	if e != nil && os.IsNotExist(e) {
		return nil, errors.Error("index.LoadCard: Card does not exist for oid %s",
			oid.Fingerprint())
	} else if e != nil {
		return nil, errors.ErrorWithCause(e, "index.LoadCard: unexpected error")
	}

	// we're always reading and immediately closing
	file, e := os.OpenFile(filename, os.O_RDONLY, system.FilePerm)
	if e != nil {
		return nil, errors.ErrorWithCause(e, "index.LoadCard: on open - unexpected error")
	}
	defer file.Close()

	size := int(finfo.Size())
	fd := int(file.Fd())
	buf, e := syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if e != nil {
		return nil, errors.ErrorWithCause(e, "index.LoadCard: on mmap - unexpected error")
	}
	defer syscall.Munmap(buf)

	/// decode base card file ///////////////////////////////////////

	var header = &cardFileHeader{}
	var offset int

	if e := header.decode(buf); e != nil {
		return nil, errors.ErrorWithCause(e, "index.LoadCard: header decode")
	}
	offset += cardHeaderSize

	var cardbase = &cardFile{
		header:   header,
		tags:     make(map[string]struct{}, header.tagcnt),
		oid:      oid,
		source:   filename,
		modified: false,
	}

	tagspec := string(buf[offset : offset+int(header.tagslen)-1])
	for _, tag := range strings.Split(tagspec, ",") {
		cardbase.tags[tag] = struct{}{}
	}
	offset += int(header.tagslen)

	/// decode typed card data //////////////////////////////////////

	var card Card
	switch cardbase.header.otype {
	case system.Text:
		tcard := &textCard{
			cardFile: cardbase,
		}
		cardbase.encode = tcard.encode
		e = tcard.decode(buf[offset:])
		card = tcard
	case system.File:
		tcard := &fileCard{
			cardFile: cardbase,
			paths:    NewPaths(),
		}
		cardbase.encode = tcard.encode
		e = tcard.decode(buf[offset:])
		card = tcard
	default:
		panic(errors.Bug("index.LoadCard: unexpected otype: %s", cardbase.header.otype))
	}

	return card, e
}

func (c *cardFile) save() (bool, error) {
	if !c.modified {
		return false, nil
	}
	if c.header.key < 0 {
		return false, errors.Bug("cardFile.save: invalid key: %d", c.header.key)
	}

	// create card dir if required
	if c.source == "" {
		if cardExists(c.oid) {
			return false, errors.Bug("cardFile.save: source is nil for existing card")
		}
		c.source = cardFilename(c.oid)
		dir := filepath.Dir(c.source)
		if e := os.MkdirAll(dir, system.DirPerm); e != nil {
			return false, errors.Bug("cardFile.save: os.Mkdirall: %s", e)
		}
	}

	/// create swapfile & mmap it /////////////////////////////////////////////////

	swapfile := fs.SwapfileName(c.source)
	sfile, _, e := fs.OpenNewSwapfile(swapfile, true)
	if e != nil {
		return false, errors.Error("cardFile.save: fs.OpenNewSwapFile: %s", e)
	}
	defer os.Remove(swapfile)
	defer sfile.Close()

	// write header and get length
	var bufsize = int64(cardHeaderSize+c.header.tagslen) + c.datalen

	if e := sfile.Truncate(bufsize); e != nil {
		return false, errors.Error("cardFile.save: file.Truncate(%d): %s", bufsize, e)
	}

	var fd = int(sfile.Fd())
	buf, e := syscall.Mmap(fd, 0, int(bufsize), syscall.PROT_WRITE, syscall.MAP_SHARED)
	if e != nil {
		return false, errors.Error("cardFile.save: syscall.Mmap: %s", e)
	}
	defer syscall.Munmap(buf)

	/// encode card ///////////////////////////////////////////////////////////////

	// REVU why not just store the trailing , and get rid of all the edge case
	//      handling in addTag/removeTag and here writing tags first ?..
	var offset = cardHeaderSize
	for tag, _ := range c.tags {
		copy(buf[offset:], []byte(tag))
		offset += len(tag)
		buf[offset] = ','
		offset++
	}
	if e := c.encode(buf[cardHeaderSize+c.header.tagslen:]); e != nil {
		return false, errors.Error("cardFile.save: encode: %s", e)
	}
	// NOTE encode header after we have all the buf encoded. (cf. header.crc32)
	if e := c.header.encode(buf); e != nil {
		return false, errors.Error("cardFile.save: header.encode: %s", e)
	}

	/// swap //////////////////////////////////////////////////////////////////////

	if e := os.Rename(swapfile, c.source); e != nil {
		return false, errors.Error("cardFile.save: os.Rename: %s", e)
	}

	c.modified = false
	return true, nil
}

/// TextCard support ///////////////////////////////////////////////////////////

type textCard struct {
	*cardFile
	text string
}

// TextCard interface defines an index.Card
type TextCard interface {
	Card
	Text() string
}

// REVU oid can be directly computed from the text.
func NewTextCard(oid *system.Oid, text string) (*textCard, error) {
	cardFile, e := newCardFile(oid, system.Text)
	if e != nil {
		return nil, e
	}
	card := &textCard{
		cardFile: cardFile,
		text:     text,
	}
	cardFile.datalen = int64(len(text))
	cardFile.encode = card.encode

	return card, nil
}

func (c *textCard) decode(buf []byte) error {
	// we're just reading a string
	c.datalen = int64(len(buf))
	var sb = make([]byte, c.datalen)
	copy(sb, buf)
	c.text = string(sb)
	return nil
}

func (c *textCard) encode(buf []byte) error {
	if len(buf) < len(c.text) {
		return errors.InvalidArg("textCard.encode", "len(buf)", "len(c.text)")
	}
	copy(buf, []byte(c.text))
	return nil
}

func (c *textCard) Text() string {
	return string(c.text)
}

func (c *textCard) Print(w io.Writer) {
	c.cardFile.Print(w)
	fmt.Fprintf(w, "text-len:  %d (debug)\n", len(c.text))
	fmt.Fprintf(w, "text:      %q\n", c.text)
	fmt.Fprintf(w, "------------------------\n\n")
}

/// FileCard support ///////////////////////////////////////////////////////////

type fileCard struct {
	*cardFile
	paths *Paths
}

type FileCard interface {
	Card
	Paths() []string
	addPath(string) (bool, error)
	removePath(string) (bool, error)
}

// REVU oid can be directly computed from the path.
func NewFileCard(oid *system.Oid, path string) (*fileCard, error) {
	cardFile, e := newCardFile(oid, system.File)
	if e != nil {
		return nil, e
	}
	paths := NewPaths()
	paths.Add(path)
	card := &fileCard{
		cardFile: cardFile,
		paths:    paths,
	}
	cardFile.datalen = int64(paths.Buflen())
	cardFile.encode = card.encode

	return card, nil
}

func (c *fileCard) encode(buf []byte) error {
	return c.paths.Encode(buf)
}

func (c *fileCard) decode(buf []byte) error {
	c.datalen = int64(len(buf))
	return c.paths.Decode(buf)
}

func (c *fileCard) Print(w io.Writer) {
	c.cardFile.Print(w)
	c.paths.Print(w)
	fmt.Fprintf(w, "------------------------\n\n")
}

func (c *fileCard) Paths() []string {
	return c.paths.List()
}

func (c *fileCard) addPath(path string) (bool, error) {
	ok, e := c.paths.Add(path)
	if ok {
		c.cardFile.datalen = int64(c.paths.Buflen())
		c.onUpdate()
	}
	return ok, e
}

func (c *fileCard) removePath(path string) (bool, error) {
	ok, e := c.paths.Remove(path)
	if ok {
		c.cardFile.datalen = int64(c.paths.Buflen())
		c.onUpdate()
	}
	return ok, e
}
