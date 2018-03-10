// Doost!

package card

import (
	//	"fmt"
	"time"

	//	"github.com/alphazero/gart/digest"
	//	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/index"
)

/// index.Card support types ///////////////////////////////////////////////////

// card file header related consts
const (
	headerBytes    = 64         // minimum length of a card file
	card_file_code = 0xec5a011e // sha256("tagmap-file")[:8]
)

// Every card has a fixed width binary header of 64 bytes
type header struct {
	ftype   uint32
	flags   uint32
	oid     [32]byte
	created int64 // unix seconds not nanos
	updated int64 // unix seconds not nanos
	crc32   uint32

	// NOTE if we don't store the oid[:8] in the oid-tags file per one option,
	//      then we need to store an offset into the oid-tags file here.
	//      At the benefit of saving 8 Bytes/entry (which can add up both in
	//      terms of size and read/write bandwidth on scans) the cost of keeping
	//      a Card file in sync with the index is incurred. It just seems too
	//      fragile.
	//
	//      Should we do so, however, then the header needs to store an offset
	//      (8 bytes) and then header size becomes 70 Bytes...)
}

// card type supports the index.Card interface. It has a fixed width header
// and a variable number of associated paths and tags.
// Not all elements of this structure are persisted in the binary image.
type card_t struct {
	header           // serialized
	dtagBah []byte   // serialized day tag' bah bitmap - write once
	stagBah []byte   // serialized systemic tags' bah bitmap - write once
	utagBah []byte   // serialized user tags' bah bitmap - can change
	paths   []string // serialized associated fs object paths

	modified bool // REVU on add/del tags, add/del paths
}

/// life-cycle ops /////////////////////////////////////////////////////////////

// Card files are created and occasionally modified. the read/write pattern is
// expected to be a quick load, read, and then possibly update and sync.
//
// Like tag.tagmap_t, on updates card_t will first write to a swap file and then
// replace its source file with the updated card data.

func Exists(oid []byte) bool {
	panic("card_t.Exists: not implemented")
}

// REVU: New needs a few input args.
func New(fname string, oid []byte, filename string, dtBah, stBah, utBah []byte) (index.Card, error) {
	panic("card_t.New: not implemented")
}

// Read an existing card file. Use card.New
func Read(fname string) (index.Card, error) {
	panic("card_t.Read: not implemented")
}

/// interface: index.Card /////////////////////////////////////////////////////
func (c *card_t) CreatedOn() time.Time { panic("cart_t: index.Card method not implemented") }
func (c *card_t) UpdatedOn() time.Time { panic("cart_t: index.Card method not implemented") }
func (c *card_t) Flags() uint32        { panic("cart_t: index.Card method not implemented") }
func (c *card_t) Oid() [32]byte        { panic("cart_t: index.Card method not implemented") }
func (c *card_t) UserTagBah() []byte   { panic("cart_t: index.Card method not implemented") }
func (c *card_t) SystemicTags() []byte { panic("cart_t: index.Card method not implemented") }
func (c *card_t) DayTagBah() []byte    { panic("cart_t: index.Card method not implemented") }
func (c *card_t) Paths() []string      { panic("cart_t: index.Card method not implemented") }
func (c *card_t) AddPath(fpath string) (bool, error) {
	panic("cart_t: index.Card method not implemented")
}
func (c *card_t) RemovePath(fpath string) (bool, error) {
	panic("cart_t: index.Card method not implemented")
}
func (c *card_t) UpdateUserTagBah(bitmap []byte) { panic("cart_t: index.Card method not implemented") }
func (c *card_t) Sync() (bool, error)            { panic("cart_t: index.Card method not implemented") }
