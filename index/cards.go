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

	"github.com/alphazero/gart/repo"
	"github.com/alphazero/gart/syslib/debug"
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
	Debug()
	Tags() []string

	setKey(int64) error               // index use only
	addTag(tag ...string) []string    // returns updated tags, if any
	removeTag(tag ...string) []string // returns removed tags, if any
	isModified() bool
	save() (bool, error) // index use only

	markDeleted() bool // returns false if locked
	IsDeleted() bool
	markLocked()
	IsLocked() bool
}

const cardHeaderSize = 40

// flags
const (
	cardDeleted byte = 1 << iota
	cardLocked
)

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
	fmt.Fprintf(w, "type:      %s\n", h.otype)
	fmt.Fprintf(w, "flags:     %08b\n", h.flags)
}
func (h *cardFileHeader) Debug() {
	w := debug.Writer
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
	fmt.Fprintf(w, "oid:       %s", c.oid.Fingerprint())
	if c.IsDeleted() {
		fmt.Fprintf(w, " deleted")
	}
	if c.IsLocked() {
		fmt.Fprintf(w, " locked")
	}
	fmt.Fprintf(w, "\n")

	if len(c.tags) > 0 {
		fmt.Fprintf(w, "tags:        \n")
		tags := c.Tags() // this sorts them
		for n, tag := range tags {
			fmt.Fprintf(w, "\t [%d]:   %q\n", n, tag)
		}
	}
}

func (c *cardFile) Debug() {
	w := debug.Writer
	fmt.Fprintf(w, "--- card ---------------\n")
	c.header.Debug()
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
	return filepath.Join(repo.IndexCardsPath, oidstr[:2], oidstr[2:])
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
	var err = errors.For("cardFile.setKey")
	if c.header.key != -1 {
		return err.Bug("key is already set to %d", c.header.key)
	}
	if key < 0 {
		return err.InvalidArg("key is < 0")
	}

	c.header.key = key
	c.onUpdate()
	return nil
}

func (c *cardFile) isModified() bool { return c.modified }

// marks card as deleted if not marked as locked.
// Returns true if card was not locked.
// REVU this is fine but it needs to be applied to setKey, etc. as well TODO
func (c *cardFile) markDeleted() bool {
	if c.header.flags&cardLocked == 0 {
		c.header.flags |= cardDeleted
		c.onUpdate()
		return true
	}
	return false
}
func (c *cardFile) IsDeleted() bool { return c.header.flags&cardDeleted != 0 }
func (c *cardFile) markLocked()     { c.header.flags |= cardLocked }
func (c *cardFile) IsLocked() bool  { return c.header.flags&cardLocked != 0 }

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
		// REVU this shoudn't be necessary anymore since we keep the
		// trailing ,
		if c.header.tagslen < 0 { // edge case of removing the only tag
			panic("why?") // TODO test this case and remove
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

func FindCard(oidstr string) ([]Card, error) {
	var err = errors.For("index.FindCard")
	if oidstr == "" {
		return nil, err.InvalidArg("oidstr is zero-len")
	}
	if len(oidstr) < 3 {
		return nil, err.InvalidArg("oidstr len:%d", len(oidstr))
	}
	var pattern string = filepath.Join(repo.IndexCardsPath, oidstr[:2], oidstr[2:])
	if len(oidstr) < system.OidSize*2 {
		pattern += "*"
	}

	files, e := filepath.Glob(pattern)
	if e != nil {
		return nil, err.ErrorWithCause(e, "on Glob(%s)", pattern)
	}

	var cards = make([]Card, len(files))
	for i, f := range files {
		debug.Printf("file:%s", f)
		oid, e := system.ParseOid(oidstr[:2] + filepath.Base(f))
		if e != nil {
			return cards, err.Bug("unexpected - %s", e)
		}
		card, e := LoadCard(oid)
		if e != nil {
			return cards, err.Bug("unexpected - %s", e)
		}
		cards[i] = card
	}
	return cards, nil
}

func LoadCard(oid *system.Oid) (Card, error) {
	var err = errors.For("index.LoadCard")

	if oid == nil {
		return nil, err.InvalidArg("oid is nil")
	}

	/// open file and map it ////////////////////////////////////////

	filename := cardFilename(oid)
	finfo, e := os.Stat(filename)
	if e != nil && os.IsNotExist(e) {
		return nil, err.Error("Card does not exist for oid %s",
			oid.Fingerprint())
	} else if e != nil {
		return nil, err.ErrorWithCause(e, "unexpected")
	}

	// we're always reading and immediately closing
	file, e := os.OpenFile(filename, os.O_RDONLY, repo.FilePerm)
	if e != nil {
		return nil, err.ErrorWithCause(e, "on open - unexpected")
	}
	defer file.Close()

	size := int(finfo.Size())
	fd := int(file.Fd())
	buf, e := syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if e != nil {
		return nil, err.ErrorWithCause(e, "on mmap - unexpected")
	}
	defer syscall.Munmap(buf)

	/// decode base card file ///////////////////////////////////////

	var header = &cardFileHeader{}
	var offset int

	if e := header.decode(buf); e != nil {
		return nil, err.ErrorWithCause(e, "header decode")
	}
	offset += cardHeaderSize

	var cardbase = &cardFile{
		header:   header,
		tags:     make(map[string]struct{}, header.tagcnt),
		oid:      oid,
		source:   filename,
		modified: false,
	}

	if header.tagcnt > 0 {
		tagspec := string(buf[offset : offset+int(header.tagslen)-1])
		for _, tag := range strings.Split(tagspec, ",") {
			cardbase.tags[tag] = struct{}{}
		}
		offset += int(header.tagslen)
	} else if header.tagslen > 0 {
		panic(err.Bug("header.tagcnt:%d - header.tagslen:%d", header.tagcnt, header.tagslen))
	}

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
		panic(err.Bug("unexpected otype: %s", cardbase.header.otype))
	}

	return card, e
}

func (c *cardFile) save() (bool, error) {
	var err = errors.For("cardFile.save")

	if !c.modified {
		return false, nil
	}
	if c.header.key < 0 {
		return false, err.Bug("invalid key: %d", c.header.key)
	}

	// create card dir if required
	if c.source == "" {
		if cardExists(c.oid) {
			return false, err.Bug("source is nil for existing card")
		}
		c.source = cardFilename(c.oid)
		dir := filepath.Dir(c.source)
		if e := os.MkdirAll(dir, repo.DirPerm); e != nil {
			return false, err.Bug("os.Mkdirall: %s", e)
		}
	}

	/// create swapfile & mmap it /////////////////////////////////////////////////

	swapfile := fs.SwapfileName(c.source)
	sfile, _, e := fs.OpenNewSwapfile(swapfile, true)
	if e != nil {
		return false, err.Error("fs.OpenNewSwapFile: %s", e)
	}
	defer os.Remove(swapfile)
	defer sfile.Close()

	// write header and get length
	var bufsize = int64(cardHeaderSize+c.header.tagslen) + c.datalen

	if e := sfile.Truncate(bufsize); e != nil {
		return false, err.Error("file.Truncate(%d): %s", bufsize, e)
	}

	var fd = int(sfile.Fd())
	buf, e := syscall.Mmap(fd, 0, int(bufsize), syscall.PROT_WRITE, syscall.MAP_SHARED)
	if e != nil {
		return false, err.Error("syscall.Mmap: %s", e)
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
		return false, err.Error("encode: %s", e)
	}
	// NOTE encode header after we have all the buf encoded. (cf. header.crc32)
	if e := c.header.encode(buf); e != nil {
		return false, err.Error("header.encode: %s", e)
	}

	/// swap //////////////////////////////////////////////////////////////////////

	if e := os.Rename(swapfile, c.source); e != nil {
		return false, err.Error("os.Rename: %s", e)
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
	var err = errors.For("textCard.encode")
	if len(buf) < len(c.text) {
		return err.InvalidArg("len(buf):%d < required %d", len(buf), len(c.text))
	}
	copy(buf, []byte(c.text))
	return nil
}

func (c *textCard) Text() string {
	return string(c.text)
}

func (c *textCard) Print(w io.Writer) {
	c.cardFile.Print(w)
	fmt.Fprintf(w, "text:      %q\n", c.text)
	fmt.Fprintf(w, "------------------------\n\n")
}
func (c *textCard) Debug() {
	c.cardFile.Debug()
	w := debug.Writer
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
func (c *fileCard) Debug() {
	c.Print(debug.Writer)
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
