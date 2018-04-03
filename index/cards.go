// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
	//	"unsafe"

	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
)

// An index.Card describes, in full, a gart object. Cards are stored as individual
// files in a .gart/index/cards/ child directory using oids for nameing the dir
// hierarchy.

type Card interface {
	Oid() *system.Oid
	Key() int64
	Type() system.Otype // REVU use for systemics tags ..
	Version() int
	Print(io.Writer)

	setKey(int64) error  // index use only
	save() (bool, error) // index use only
}

// 1       2         1       4       8         8         8
// otype / version / flags / crc32 / created / updated / key
const cardHeaderSize = 32

type cardFile struct {
	// pseudo header
	otype   system.Otype
	version int16
	flags   byte
	crc32   uint32
	created int64
	updated int64
	key     int64

	datalen  int64 // REVU is this even necessary?
	oid      *system.Oid
	source   string
	buf      []byte // from mmap
	modified bool
	encode   func([]byte) error
}

func (c *cardFile) Print(w io.Writer) {
	fmt.Fprintf(w, "--- card ---------------\n")
	fmt.Fprintf(w, "type:      %s\n", c.otype)
	fmt.Fprintf(w, "version:   %d\n", c.version)
	fmt.Fprintf(w, "flags:     %08b\n", c.flags)
	fmt.Fprintf(w, "crc32:     %08x\n", c.crc32)
	fmt.Fprintf(w, "created:   %d - %s\n", c.created, time.Unix(0, c.created))
	fmt.Fprintf(w, "updated:   %d - %s\n", c.updated, time.Unix(0, c.updated))
	fmt.Fprintf(w, "key:       %d\n", c.key)
	fmt.Fprintf(w, "oid:       %s\n", c.oid.Fingerprint())
	fmt.Fprintf(w, "source:    %q\n", cardFilename(c.oid)) // XXX c.source)
	fmt.Fprintf(w, "data-len:  %d\n", c.datalen)
	fmt.Fprintf(w, "modified:  %t\n", c.modified)
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
	card := &cardFile{
		otype:   otype,
		version: -1,
		flags:   0,
		crc32:   0,
		created: now,
		updated: 0,
		key:     -1, // REVU change these -1s to index.invalidKey (objects.go)

		oid:      oid,
		modified: false,
	}

	return card, nil
}

/// Card support ///////////////////////////////////////////////////////////////

func (c *cardFile) Oid() *system.Oid   { return c.oid }
func (c *cardFile) Key() int64         { return c.key }
func (c *cardFile) Type() system.Otype { return c.otype }
func (c *cardFile) Version() int       { return int(c.version) }
func (c *cardFile) setKey(key int64) error {
	if c.key != -1 {
		return errors.Bug("cardFile.setKey: key is already set to %d", c.key)
	}
	if key < 0 {
		return errors.InvalidArg("cardFile.setKey", "key", "< 0")
	}

	c.key = key
	c.onUpdate()
	return nil
}

func (c *cardFile) onUpdate() {
	if !c.modified {
		c.modified = true
		c.version++
		c.updated = time.Now().UnixNano()
	}
}

func (c *cardFile) debugStr() string {
	return fmt.Sprintf("%s-card:(oid:%s key:%d)", c.otype, c.oid.Fingerprint(), c.key)
}

/// io ops /////////////////////////////////////////////////////////////////////

// REVU this would have to partially create cardFile and then pass it to newTypeCard
func loadCard(oid *system.Oid) (Card, error) {
	panic(errors.NotImplemented("wip"))
}

func (c *cardFile) save() (bool, error) {
	if !c.modified {
		return false, nil
	}
	if c.key < 0 {
		return false, errors.Bug("cardFile.save: invalid key: %d", c.key)
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
	header := fmt.Sprintf("%s\n", c.otype)
	header += fmt.Sprintf("%s\n", c.oid.String())
	header += fmt.Sprintf("%d\n", c.version)
	header += fmt.Sprintf("%d\n", c.datalen)
	var headerSize = len(header)
	var bufsize = int64(headerSize) + c.datalen

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

	copy(buf, []byte(header))

	if e := c.encode(buf[headerSize:]); e != nil {
		return false, errors.Error("cardFile.save: encode: %s", e)
	}

	/// swap //////////////////////////////////////////////////////////////////////

	if e := os.Rename(swapfile, c.source); e != nil {
		return false, errors.Error("cardFile.save: os.Rename: %s", e)
	}

	c.modified = false
	return true, nil
}

func encodeFileCard(buf []byte) error { panic("do it") }

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
	system.Debugf("index.NewTextCard: oid:%s text:%q\n", oid.Fingerprint(), text)
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
	AddPath(string) (bool, error)
	RemovePath(string) (bool, error)
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

func (c *fileCard) Print(w io.Writer) {
	c.cardFile.Print(w)
	c.paths.Print(w)
	fmt.Fprintf(w, "------------------------\n\n")
}

func (c *fileCard) Paths() []string {
	return c.paths.List()
}

func (c *fileCard) AddPath(path string) (bool, error) {
	return c.paths.Add(path)
}

func (c *fileCard) RemovePath(path string) (bool, error) {
	return c.paths.Remove(path)
}
