// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
)

// An index.Card describes, in full, a gart object. Cards are stored as individual
// files in .gart/index/cards/ directory in a manner similar to git blobs. Cards
// are human readable, carriage return (\n) delimited files. (Given that typical
// host OS file system will allocate (typically) up to 4k per inode (even for a 1
// byte file) the design choice for plain-text (unicode) encoding of cards seems
// reasonable. Card files can also be compressed.
//
// Each card (redundantly) stores the object identity (Oid), associated tags (in
// plain text and not via a bitmap), systemic tags, system timestamps, and crc.
//
// Use cases for index.Card:
//
// Being the 'leaves' of the OS FS's (gratis) b-tree index structure, index.Cards
// allow for O(log) lookup of an object's details via Oid. Both individual file
// queries (i.e. gart-find -f <some-file>) and tag based queries (i.e. gart-find
// -tags <csv tag list>) internally resolve to one or more system.Oids. Given an
// Oid, access to the associated card is via os.Open.
//
// Cards are also the fundamental index recovery mechanism for gart. Given the
// set of index.Cards, the Object-index (object.idx), and associated Tag bitmaps
// can be rebuilt. Even the Tag Dictionary can be rebuilt given the set of Cards.
//
// For this reason, and given their small size, index.Cards are always read in full
// in RDONLY mode and updates are via SYNC'ed swaps.
//
// Card file format:
//
//    line  datum -- all lines are \n delimited.
//
//    0:    %016x formatted representation of CRC64 of card file lines 1->n.
//	  1:    %016x formatted object.idx key.
//    2:    0%16x %016x %d formatted create, update, and revision number.
//    3:    reserved for flags if any. this line may simply be a \n.
//    4:    %d %d formatted tag-count and initial line number for tags
//    5:    %d %d formatted path-count and initial line number for paths
//    6:    %d %d formatted systemics-count and initial line number for systemics
//    <path-count> lines are absolute path specs.
//    <tag-count> lines are tag names.
//    <systemic-count> lines are systemic attributes and flags.
//
// Example:
//
//  --- begin ----------------------
//  1:	 73cb3858a687a849
//  2:	 cd777f8ec7a2743f8190f54f5c189607357a29bd86fd49f006fef81647d99dbb
//  3:	 15210c6ca746f5ad 15210c6d48ad124f 1
//  4:	 7
//  5:	 Friend, Doost, Beloved, Salaam, Samad, Sultan, LOVE
//  6:	 2
//  7:	 .go, mar-31-2018
//  8:	 2
//  9:	 /Users/alphazero/Code/go/src/gart/index/ftest/test-index.go
//  10:	 /Volumes/OpenGate/Backups/ove/alphazero/Code/go/src/gart/index/ftest/test-index.go
//  --- end ------------------------
//
// REVU	each card load has to decode crc and timestamps. it also needs to decode
//		the counts. The rest are plain text in the binary encoded version as well.
//      The only difference, really, from 1.0 version is that we no longer use a BAH
//      and have no limits on number of tags. (before the BAH had to be 255 bytes
// 		max and of course since it was a bitmap, we needs tag.dict file to recover.)
//
//		One argument for plain-text is that it is 'human readable', but counter arg
//		is that there will be a gart-info -oid to decode it.
//
//		Parsing will not be faster or simpler. We still have to chase the \n terminal.
//
//		Parsing binary will be faster. It is true that it will be 'noise' for one
// 		card read, but still saving piping find . to gart-add will add up those
//		incremental +deltas.
//
//	    Reminder that the binary form will have <path-len><path-in-plaintext>, etc.
//		so parsing is deterministic.
//
// TODO sleep on this.
//
// REVU the simplest thing that would work:
//
//		a card simply is:
//		object type : string : in { blob, file }
//		object key : int : in [0, n]
//		version : int : in [1, n]
//		-- type specific data formats
//		list of paths for file objects
//		or
//		embedded blob
//
//		and that's it.

type Card interface {
	Oid() *system.Oid
	Key() int64
	Type() system.Otype // REVU use for systemics tags ..
	Version() int
	Print(io.Writer)

	setKey(int64) error  // index use only
	save() (bool, error) // index use only
}

// abcd1234/data/1
// data: len/bytes/crc
type cardFile struct {
	oid *system.Oid

	key     int64
	otype   system.Otype
	version int
	datalen uint64 // REVU is this even necessary?
	datacrc uint64 // REVU this is now problematic

	source string
	buf    []byte // from mmap

	modified bool
	encode   func([]byte) error
}

func (c *cardFile) Print(w io.Writer) {
	fmt.Fprintf(w, "card-type: %s\n", c.otype)
	fmt.Fprintf(w, "oid:       %s\n", c.oid.Fingerprint())
	fmt.Fprintf(w, "key:       %d\n", c.key)
	fmt.Fprintf(w, "version:   %d\n", c.version)
	fmt.Fprintf(w, "data-len:  %d\n", c.datalen)
	fmt.Fprintf(w, "data-crc:  %d\n", c.datacrc)
	fmt.Fprintf(w, "source:    %q\n", cardFilename(c.oid)) // XXX c.source)
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

	card := &cardFile{
		oid:     oid,
		key:     -1,
		otype:   otype,
		version: 1,
		//		datalen: uint64(len(data)), // REVU important extensions must provide datalen()
		modified: true,
	}

	return card, nil
}

/// Card support ///////////////////////////////////////////////////////////////

func (c *cardFile) Oid() *system.Oid   { return c.oid }
func (c *cardFile) Key() int64         { return c.key }
func (c *cardFile) Type() system.Otype { return c.otype }
func (c *cardFile) Version() int       { return c.version }
func (c *cardFile) setKey(key int64) error {
	if c.key != -1 {
		return errors.Bug("cardFile.setKey: key is already set to %d", c.key)
	}
	if key < 0 {
		return errors.InvalidArg("cardFile.setKey", "key", "< 0")
	}
	c.key = key
	return nil
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
	system.Debugf("cardFile.save: %s - BEGIN\n", c.debugStr())

	// check state validity

	if !c.modified {
		system.Debugf("cardFile.save: card is not modified\n")
		return false, nil
	}
	if c.key < 0 {
		return false, errors.Bug("cardFile.save: invalid key: %d", c.key)
	}

	// create card dir if required

	if c.source == "" {
		system.Debugf("cardFile.save: source not defined - assume newCard\n")
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
	var bufsize = int64(headerSize)

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
	cardFile.datalen = uint64(len(text))
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
	cardFile.datalen = uint64(paths.Buflen())
	cardFile.encode = card.encode

	return card, nil
}

func (c *fileCard) encode(buf []byte) error {
	return c.paths.Encode(buf)
}

func (c *fileCard) Print(w io.Writer) {
	c.cardFile.Print(w)
	c.paths.Print(w)
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
