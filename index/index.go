// Doost!

package index

import (
	"fmt"
	"time"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/digest"
)

/// Index Cards ////////////////////////////////////////////////////////////////

// Card defines the interface of an Index-Card.
type Card interface {
	CreatedOn() time.Time // unix seconds precision
	UpdatedOn() time.Time // unix seconds precision
	Flags() byte          // REVU use semantic flags e.g. Card.HasDups() bool, etc.
	Revision() int        // 0 indicates new card

	Oid() OID

	Tags() bitmap.Bitmap
	SetTags(cpm bitmap.Bitmap) error
	UpdateTags(cpm bitmap.Bitmap) (bool, error)

	Systemic() bitmap.Bitmap
	SetSystemics(cpm bitmap.Bitmap) error
	UpdateSystemics(cpm bitmap.Bitmap) (bool, error)

	Paths() []string // REVU len(card.Paths()) > 1 => dup files
	AddPath(fpath string) (bool, error)
	RemovePath(fpath string) (bool, error)

	Save() (bool, error)

	DebugStr() string
}

const (
	// key value for unindexed cards
	notIndexed = uint64(0xffffffffffffffff)
)

// indexedCard is a package private interface encapsulating card indexing related
// features that should not be exposed via Card.
type indexedCard interface {
	// returns 64bit index key for card
	Key() uint64
	// sets the index key for the card
	SetKey(uint64)
}

// Cards defines the interface of an Index-Card manager. It is used to isolate
// the index ops from changes in the implementation of Card persistence impl.
// It is a sort of entity mamanager.
type Cards interface {
	// REVU card life-cycle per gart operations. Below is tentative as of now.
}

/// Object IDs /////////////////////////////////////////////////////////////////

const (
	oidBytesLen = 32
)

// export the type but keep internals private to index package
type OID struct {
	dat [oidBytesLen]byte
}

// REVU: this should be the only way to get an OID
func ObjectId(fpath string) (*OID, error) {

	md, e := digest.SumFile(fpath)
	if e != nil {
		return nil, fmt.Errorf("index.ObjectId: digest.SumFile: err: %s", e)
	}

	return newOid(md), nil
}

// internal func panics on errors
func newOid(bytes []byte) *OID {

	if bug := validateOidBytes(bytes); bug != nil {
		panic(fmt.Errorf("bug - index.newOid: invalid arg - %s", bug))
	}
	var oid OID
	copy(oid.dat[:], bytes[:oidBytesLen])

	return &oid
}

func validateOidBytes(bytes []byte) error {
	if len(bytes) < oidBytesLen {
		return fmt.Errorf("bug - invalid OID bytes - len: %d", len(bytes))
	}
	for _, b := range bytes {
		if b != 0x00 {
			return nil
		}
	}
	return fmt.Errorf("bug - invalid OID bytes - all 0x00")
}

func (this *OID) isEqual(that *OID) bool {
	for i := 0; i < oidBytesLen; i++ {
		if this.dat[i] != that.dat[i] {
			return false
		}
	}
	return true
}

func (oid *OID) String() string { return fmt.Sprintf("%x", oid.dat) }

/// Card ops ///////////////////////////////////////////////////////////////////

var (
	ErrCardExists   = fmt.Errorf("index.Card: card exists.")
	ErrCardNotFound = fmt.Errorf("index.Card: card for oid not found.")
)

// returns (Card, newCard, updated, e)
// Card is always saved on success.
func AddOrUpdateCard(garthome string, oid *OID, file string, tbm, sbm bitmap.Bitmap) (Card, bool, bool, error) {

	card, newCard, e := getOrCreateCard(garthome, oid)
	if e != nil {
		return nil, false, false, e
	}

	var rev0 = card.Revision()

	if ok, e := card.AddPath(file); e != nil {
		return nil, false, false, e
	} else if newCard && !ok {
		err := fmt.Errorf("bug - index.AddOrUpdateCard: path not added on new card.")
		return nil, false, false, err
	}

	if e := card.SetTags(tbm); e != nil {
		return nil, false, false, e
	}

	if e := card.SetSystemics(sbm); e != nil {
		return nil, false, false, e
	}

	var updated = card.Revision() > rev0
	if ok, e := card.Save(); e != nil {
		return nil, false, false, e
	} else if newCard && !ok {
		err := fmt.Errorf("bug - index.AddOrUpdateCard: card.Save not ok new card.")
		return nil, false, false, err
	} else if updated && !ok {
		err := fmt.Errorf("bug - index.AddOrUpdateCard: card.Save not ok on revision change.")
		return nil, false, false, err
	}
	return card, newCard, updated, nil
}

// Read only - gets the card.
// Returns (nil, ErrCardNotFound) if card not found.
func GetCard(garthome string, oid *OID) (Card, error) {
	card, e := readCard(garthome, oid) // HERE this should be via Cards interface
	if e != nil && e != ErrCardNotFound {
		return nil, fmt.Errorf("err - index.GetCard: unexpected error - %v", e)
	}
	return card, e
}

// either gets existing card or creats a new card.
// returns (Card, newCard, e)
// TODO is to use Cards interface to create cards.
func getOrCreateCard(garthome string, oid *OID) (Card, bool, error) {

	card, e := GetCard(garthome, oid)
	if e != nil && e == ErrCardNotFound {
		card, e := newCard(garthome, oid, notIndexed)
		return card, true, e
	} else if e != nil {
		return nil, false, e
	}
	return card, false, e
}
