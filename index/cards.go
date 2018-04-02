// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alphazero/gart/syslib/errors"
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
	Save() (bool, error)
	Print(io.Writer)
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

func newCardFile(oid *system.Oid, otype system.Otype, key int64) (*cardFile, error) {
	if e := otype.Verify(); e != nil {
		return nil, e
	}
	if key < 0 {
		return nil, errors.InvalidArg("index.newCard", "key", "< 0")
	}

	card := &cardFile{
		oid:     oid,
		key:     key,
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

// REVU this would have to partially create cardFile and then pass it to newTypeCard
func loadCard(oid *system.Oid) (Card, error) { panic(errors.NotImplemented("wip")) }

// REVU this would have to override encode(buf) of cardFile and write data
func (c *textCard) Save() (bool, error) {
	panic(errors.NotImplemented("wip"))
}

// TODO start here ..
// REVU this would have to override encode(buf) of cardFile and invoke paths.encode
func (c *fileCard) Save() (bool, error) { panic(errors.NotImplemented("wip")) }

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
func NewTextCard(oid *system.Oid, key int64, text string) (*textCard, error) {
	cardFile, e := newCardFile(oid, system.Text, key)
	if e != nil {
		return nil, e
	}
	cardFile.datalen = uint64(len(text))

	card := &textCard{
		cardFile: cardFile,
		text:     text,
	}
	return card, nil
}

func (c *textCard) Text() string {
	return string(c.text)
}

func (c *textCard) Print(w io.Writer) {
	fmt.Fprintf(w, "textCard: not what we want! :)\n")
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
func NewFileCard(oid *system.Oid, key int64, path string) (*fileCard, error) {
	cardFile, e := newCardFile(oid, system.Text, key)
	if e != nil {
		return nil, e
	}
	paths := NewPaths()
	paths.Add(path)
	cardFile.datalen = uint64(paths.Buflen())

	card := &fileCard{
		cardFile: cardFile,
		paths:    paths,
	}
	return card, nil
}

func (c *fileCard) Print(w io.Writer) {
	fmt.Fprintf(w, "fileCard: not what we want! :)\n")
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
