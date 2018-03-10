// Doost!

package index

import (
	"time"
)

/// interfaces /////////////////////////////////////////////////////////////////

// An index card
type Card interface {
	CreatedOn() time.Time // unix seconds precision
	UpdatedOn() time.Time // unix seconds precision
	Flags() uint32        // REVU: use semantic flags e.g. Card.HasDups() bool, etc.

	// Object ID is the Card entry's content hash
	Oid() [32]byte
	//
	UserTagBah() []byte
	//
	SystemicTags() []byte
	//
	DayTagBah() []byte
	//
	Paths() []string
	//
	AddPath(fpath string) (bool, error)
	//
	RemovePath(fpath string) (bool, error)
	//
	UpdateUserTagBah(bitmap []byte)
	//
	// Sync updates the persistent image
	Sync() (bool, error)
}
