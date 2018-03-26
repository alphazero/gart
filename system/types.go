// Doost!

package system

import (
	"fmt"
	"time"

	"github.com/alphazero/gart/syslib/errors"
)

/// Runtime Context ////////////////////////////////////////////////////////////

// Context is the system runtime context.
type Context struct {
	TagManager
	Index
}

/// Object tagging /////////////////////////////////////////////////////////////

// A Tag is a user defined attribute associated with archived content. Tags
// unique (but case insensitive) user defined names and system assigned ids.
type Tag interface {
	// Returns the user defined name of the Tag.
	Name() string
	// Returns the non-zero unique system id assigned to this Tag.
	Id() int
	// Returns the number of objects tagged with this Tag.
	Refcnt() int
}

// TagManager defines the interface to a persistent store of user defined
// Tags, allowing for the addtion, lookup, and updating of Tag reference counts.
type TagManager interface {
	// Returns the number of Tags in the dictionary.
	Size() int

	// Adds a new tag. Tag names are case-insensitive, non-zerolen, and at most
	// MaxTagNameSize. An ErrInvalidTagName error is returned
	//
	// Returns (true, id, nil) if tag was added, otherwise (false, id, nil)
	// with the id of the existing tag.
	//
	// Returns (false, 0, ErrInvalidArg) if the size requirement is not
	// met. Any other error returned is indicating of a bug or fault.
	Add(name string) (added bool, id int, err error)

	// Increments the named tag's refcnt and returns the new refcnt.
	// Returns ErrTagNotFound error if tag does not exist. Any other error
	// is indicative of a bug or fault.
	IncrRefcnt(name string) (refcnt int, id int, err error)

	// Returns ids of selected tags. These are used to build index bitmaps.
	// notDefined is never nil. If not empty, it contains all
	// tag names that are not defined.
	SelectTags(names []string) (ids []int, notDefined []string)

	// Syncs the tagmap file. IFF the in-mem model has been modified
	Sync() (ok bool, e error)

	// List tags
	Tags() []Tag
}

/// Object indexing ////////////////////////////////////////////////////////////

// Card defines the public attributes of the index card of an archived object.
type IndexCard interface {
	CreatedOn() time.Time // unix seconds precision
	UpdatedOn() time.Time // unix seconds precision
	Revision() int        // 0 indicates new card
	ObjectId() *Oid       //
	Tags() []string       //
	Systemic() []string   //
	Paths() []string      // REVU len(card.Paths()) > 1 => dup files

	// REVU see 1.0/index.go#indexCard interface. For providers.
	//	IndexKey() uint64     // 0 indicates card is not indexed

	// REVU move to Index ?
	//	AddPath(fpath string) (bool, error)
	//	RemovePath(fpath string) (bool, error)
	//	SetTags(cpm bitmap.Bitmap) (bool, error)
	//	SetSystemics(cpm bitmap.Bitmap) (bool, error)
	//	Save() (bool, error)
	//	DebugStr() string
}

type Index interface {
	/* AddObject, Lookup, .. ? */
}

/// Object Identity ////////////////////////////////////////////////////////////

// Object Identity is used to uniquely identity an archived content.
type Oid struct {
	dat [OidSize]byte
}

// NewOid expects a slice of atleast OidSize bytes, which will be validated
// and then copied for the allocated Oid.
//
// As this is a system function, all errors are treated as bugs.
//
// Returns (nil, BugInvalidArg) if the slice length is not as specified.
// Returns (nil, BugInvalidOidBytesData) if the minimal validation fails.
func NewOid(bytes []byte) (*Oid, error) {
	if len(bytes) < OidSize {
		return nil, errors.BugInvalidArg
	}
	// sans actual content to verify, the minimal validation of the data
	// is that the slice can not possibly be all 0x00
	for _, b := range bytes {
		if b != 0x00 {
			goto valid
		}
	}
	// if reached, it was all 0x00
	return nil, BugInvalidOidBytesData

valid:
	var oid Oid
	copy(oid.dat[:], bytes[:OidSize])
	return &oid, nil
}

func (oid *Oid) String() string { return fmt.Sprintf("Oid:%x", oid.dat) }
