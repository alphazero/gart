// Doost!

package index

import (
	"fmt"
	"io"

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
	Type() system.Otype
	Version() int
	Save() (bool, error)
	Print(io.Writer)
}

// abcd1234/data/1
// data: len/bytes/crc
type cardFile struct {
	key     int64
	otype   system.Otype
	version int
	datalen uint64 // REVU this is now problematic
	datacrc uint64 // REVU this is now problematic
	//	data    []byte // read from file, also used for text
	//	paths   *Paths // non-nil if otype == system.File

	oid      *system.Oid
	modified bool
}

/// cardFile ///////////////////////////////////////////////////////////////////

func newCardFile(oid *system.Oid, otype system.Otype, key int64) (*cardFile, error) {
	if e := otype.Verify(); e != nil {
		return nil, e
	}
	if key < 0 {
		return nil, errors.InvalidArg("index.newCard", "key", "< 0")
	}

	/* XXX deprecated?
	var textdata []byte
	var paths *Paths
	switch otype {
	case system.Text:
		textdata = data
	case system.File:
		paths = NewPaths()
		if e := paths.Decode(data); e != nil {
			return nil, errors.ErrorWithCause(e, "index.newCard: on newPaths(data)")
		}
	case system.URL, system.URI:
		return nil, errors.NotImplemented("index.newCard: card type:%s", otype)
	}
	*/
	card := &cardFile{
		oid:     oid,
		key:     key,
		otype:   otype,
		version: 1,
		//		datalen: uint64(len(data)), // REVU important extensions must provide datalen()
		//		data:     textdata,
		//		paths:    paths,
		modified: true,
	}

	// TODO create oid based dir/filename for card.

	return card, nil
}

/*
func (c *cardFile) Print(w io.Writer) {
	panic(errors.Bug("cardFile.Print: type %s not supported", c.otype))
	switch c.otype {
	case system.Text:
		c.textCardPrint(w)
	case system.File:
		c.fileCardPrint(w)
	default:
		panic(errors.Bug("cardFile.Print: type %s not supported", c.otype))
	}
}
*/
/// Card support ///////////////////////////////////////////////////////////////

func (c *cardFile) Oid() *system.Oid   { return c.oid }
func (c *cardFile) Key() int64         { return c.key }
func (c *cardFile) Type() system.Otype { return c.otype }
func (c *cardFile) Version() int       { return c.version }

// panics if cardFile object type does not match the otype arg.
//func (c *cardFile) assertType(otype system.Otype) {
//	if c.otype != system.Text {
//		panic(errors.Bug("cardFile.assertType: otype is %s not %s", c.otype, otype))
//	}
//}

// REVU this would have to partially create cardFile and then pass it to newTypeCard
func loadCard(oid *system.Oid) (Card, error) { panic(errors.NotImplemented("wip")) }

// REVU this would have to override encode(buf) of cardFile and write data
func (c *textCard) Save() (bool, error) { panic(errors.NotImplemented("wip")) }

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
func NewTextCard(oid *system.Oid, key int64, text string) (TextCard, error) {
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

//func (c *cardFile) TextCard() TextCard {
//	c.assertType(system.Text)
//	return c
//}

func (c *textCard) Text() string {
	//	c.assertType(system.Text)
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
func NewFileCard(oid *system.Oid, key int64, path string) (FileCard, error) {
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

//func (c *cardFile) FileCard() FileCard {
//	c.assertType(system.File)
//	return c
//}

func (c *fileCard) Paths() []string {
	//	c.assertType(system.File)
	return c.paths.List()
}

func (c *fileCard) AddPath(path string) (bool, error) {
	//	c.assertType(system.File)
	return c.paths.Add(path)
}

func (c *fileCard) RemovePath(path string) (bool, error) {
	//	c.assertType(system.File)
	return c.paths.Remove(path)
}
