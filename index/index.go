// Doost!

package index

import (
	"fmt"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system" // TODO  REVU decision for OID in system ..
)

/// index package errors ///////////////////////////////////////////////////////

var (
	ErrObjectIndexExist    = errors.Error("%s exists", idxFilename)
	ErrObjectIndexNotExist = errors.Error("%s does not exist", idxFilename)
	ErrObjectIndexClosed   = errors.Error("object index is closeed")
)

/// types //////////////////////////////////////////////////////////////////////

// TODO
// significant todo here is nailing down the 'query' for selecting all object ids
// where tags are 'set'. Simple option is to for now just do a specific function for
// querying OIDs for a given set of tags to be ANDed.

type indexManager struct{}

func OpenIndexManager(repoDir string) (*indexManager, error) {
	var idxmgr = &indexManager{}

	// REVU for now. probably needs to have a 'tag manager' instance
	// and check various required files.
	// TODO make sure if we are -not- passing repoDir around anymore
	// to also remove it from this
	return idxmgr, nil
}

// REVU need to revisit system/types.go
// REVU this also requires a functional objects.go (object-index) to
// match the 'bit' of the ANDed tagmaps with an OID. But the general structure is
// possible to sketch out here and test (for the bits.)
//
// Returns all oids that match for given tags. array may be empty but never nil.
// Return error if any of the tags are undefined.
func (idx *indexManager) SelectObjects(tags ...string) ([]*system.Oid, error) {

	tagmap0, e := LoadTagmap(tags[0], false)
	if e != nil {
		return nil, e
	}

	var oids = []*system.Oid{} // initial empty result set

	var resmap = tagmap0.bitmap
	for _, tag := range tags[1:] {
		tagmap, e := LoadTagmap(tag, false)
		if e != nil {
			return nil, e
		}
		if resmap, e = tagmap.bitmap.And(resmap); e != nil {
			return nil, e
		}
		// if results are already an empty set just return
		if len(resmap.Bits()) == 0 {
			return oids, nil
		}
	}

	fmt.Printf("debug - selected key ids:\n")
	for _, key := range resmap.Bits() {
		fmt.Printf("key: %d\n", key)
	}

	// TODO pass keys to objects-index to get the OID list
	// oids = GetObjectIds(resmap.Bits())

	return oids, nil

}
