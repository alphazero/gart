// Doost!

package index

import (
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

// abcd1234/data/1
// data: len/bytes/crc
type cardFile struct {
	key     int64
	otype   system.Otype
	version int
	datalen uint64
	datacrc uint64
	data    []byte // read from file, also used for text
	paths   *Paths // non-nil if otype == system.File

	oid      *system.Oid
	modified bool
}

type cardbase interface {
	Oid() *system.Oid
	Key() int64
	Type() system.Otype
	Version() int
}

type TextCard interface {
	cardbase
	Text() string
}

type FileCard interface {
	cardbase
	Paths() []string
	AddPath(string) (bool, error)
	RemovePath(string) (bool, error)
}

func NewTextCard(oid *system.Oid, key int64, text string) (*cardFile, error) {
	return newCardfile(oid, system.Text, key, []byte(text))
}

func NewFileCard(oid *system.Oid, key int64, path string) (*cardFile, error) {
	return newCardfile(oid, system.Text, key, []byte(path))
}

func newCardfile(oid *system.Oid, otype system.Otype, key int64, data []byte) (*cardFile, error) {
	if e := otype.Verify(); e != nil {
		return nil, e
	}
	if key < 0 {
		return nil, errors.InvalidArg("index.newCardfile", "key", "< 0")
	}

	var textdata []byte
	var paths *Paths
	switch otype {
	case system.Text:
		textdata = data
	case system.File:
		paths = NewPaths()
		if e := paths.decode(data); e != nil {
			return nil, errors.ErrorWithCause(e, "index.newCardFile: on newPaths(data)")
		}
	case system.URL, system.URI:
		return nil, errors.NotImplemented("index.newCardFile: card type:%s", otype)
	}

	card := &cardFile{
		oid:      oid,
		key:      key,
		otype:    otype,
		version:  1,
		datalen:  uint64(len(data)),
		data:     textdata,
		paths:    paths,
		modified: true,
	}

	// TODO create oid based dir/filename for card.

	return card, nil
}

func (c *cardFile) Oid() *system.Oid   { return c.oid }
func (c *cardFile) Key() int64         { return c.key }
func (c *cardFile) Type() system.Otype { return c.otype }
func (c *cardFile) Version() int       { return c.version }

func (c *cardFile) TextCard() TextCard {
	c.assertType(system.Text)
	return c
}

func (c *cardFile) FileCard() FileCard {
	c.assertType(system.File)
	return c
}

// panics
func (c *cardFile) assertType(otype system.Otype) {
	if c.otype != system.Text {
		panic(errors.Bug("cardFile.assertType: otype is %s not %s", c.otype, otype))
	}
}

func (c *cardFile) Text() string {
	c.assertType(system.Text)
	return string(c.data)
}

func (c *cardFile) Paths() []string {
	c.assertType(system.File)
	return c.paths.arr
}

func (c *cardFile) AddPath(path string) (bool, error) {
	c.assertType(system.File)
	return c.paths.add(path)
}

func (c *cardFile) RemovePath(path string) (bool, error) {
	c.assertType(system.File)
	return c.paths.remove(path)
}

/// Paths //////////////////////////////////////////////////////////////////////

type Paths struct {
	arr []string
}

func NewPaths() *Paths {
	return &Paths{make([]string, 0)}
}
func (p *Paths) add(path string) (bool, error) {
	return false, errors.NotImplemented("Paths.add")
}

func (p *Paths) remove(path string) (bool, error) {
	return false, errors.NotImplemented("Paths.remove")
}

func (v Paths) size() int { return len(v.arr) }
func (v Paths) buflen() int {
	if len(v.arr) == 0 {
		return 0
	}
	var blen int
	for _, s := range v.arr {
		blen += len(s) + 1 // carriage-return delim
	}
	return blen
}

// REVU copies the bytes so it is safe with mmap.
// REVU exported for testing TODO doesn't need to be exported
func (p *Paths) decode(buf []byte) error {
	if buf == nil {
		return errors.InvalidArg("Paths.decode", "buf", "nil")
	}
	readLine := func(buf []byte) (int, []byte) {
		var xof int
		for xof < len(buf) {
			if buf[xof] == '\n' {
				break
			}
			xof++
		}
		return xof + 1, buf[:xof]
	}
	var xof int
	for xof < len(buf) {
		n, path := readLine(buf[xof:])
		p.arr = append(p.arr, string(path))
		xof += n
	}
	return nil
}

func (v Paths) encode(buf []byte) error {
	if buf == nil {
		return errors.InvalidArg("Paths.encode", "buf", "nil")
	}
	if len(buf) < v.buflen() {
		return errors.InvalidArg("Paths.encode", "buf", "< path.buflen")
	}
	var xof int
	for _, s := range v.arr {
		copy(buf[xof:], []byte(s))
		xof += len(s)
		buf[xof] = '\n'
		xof++
	}
	return nil
}
