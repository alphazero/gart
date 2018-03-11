// Doost!

package tag

import (
	"time"

	"github.com/alphazero/gart/fs"
)

// tag.Map is the main interface of this package with the rest of the system.
type Map interface {
	// aka Tag count
	Size() uint64

	CreatedOn() time.Time

	UpdatedOn() time.Time

	// Adds a new tag. Tag name must be at most maxNameBytes or an error is returned.
	// Further note that tag names are case-insensitive.
	// added is true if tag was indeed new. Otherwise false (with no error)
	// If error is nil, the tag id is returned.
	Add(name string) (added bool, id int, err error)

	// Increments the named tag's refcnt and returns the new refcnt.
	// returns error if tag does not exist.
	IncrRefcnt(name string) (refcnt int, id int, err error)

	// Returns ids of selected tags. These are used to construct BAH bitmaps.
	// notDefined is never nil. If not empty, it contains all
	// tag names that are not defined.
	SelectTags(names []string) (ids []int, notDefined []string)

	// Syncs the tagmap file. IFF the in-mem model has been modified
	Sync() (ok bool, e error)

	// List tags
	Tags() []Tag
}

/// gart capabilities /////////////////////////////////////////////////////////

// Updates the provided tag.Map for the gart object. This function is called by
// gart-add (only?) Remember that a gart object maps to 1 or more fs objects.
//
func UpdateMapForNewObject(tagmap Map, fsd *fs.FileDetails, tags []string) ([]int, error) {
	// systemics: add if necessary and update refcnt
	// user tags: same
	panic("tag.UpdateMapForNewObject: not implemented")
}
