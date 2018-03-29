// Doost!

package index

import (
	"fmt"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system" // TODO  REVU decision for OID in system ..
)

/// index package errors ///////////////////////////////////////////////////////

var (
	ErrObjectIndexExist    = errors.Error("%s exists", oidxFilename)
	ErrObjectIndexNotExist = errors.Error("%s does not exist", oidxFilename)
	ErrObjectIndexClosed   = errors.Error("object index is closeed")
)

/// index manager //////////////////////////////////////////////////////////////

// TODO
// significant todo here is nailing down the 'query' for selecting all object ids
// where tags are 'set'. Simple option is to for now just do a specific function for
// querying OIDs for a given set of tags to be ANDed.

// REVU be consistent with type name (exported or pkg prv.)
type indexManager struct {
	opMode  OpMode
	oidx    *oidxFile
	tagmaps map[string]*Tagmap
}

// index.Initialize creates the various necessary indexes for gart in the
// canonical system.RepoDir.
func InitializeRepo() error {
	if e := createObjectIndex(); e != nil {
		return errors.ErrorWithCause(e,
			"index.InitializeRepo: error creating objects index")
	}

	// REVU what else?
	//		Tagmaps are per tag. Should it create Tagdict?
	//		TODO tagmaps for 'systemic' tags could be done here.
	//		Card files are per object. Is there to be a Cards object?

	return nil
}

// REVU: turns out OpMode is not just for objects.go.
func OpenIndexManager(opMode OpMode) (*indexManager, error) {

	oidx, e := openObjectIndex(opMode)
	if e != nil {
		return nil, e
	}

	var idxmgr = &indexManager{
		opMode:  opMode,
		oidx:    oidx,
		tagmaps: make(map[string]*Tagmap),
	}

	return idxmgr, nil
}

// Preloads the associated Tagmaps for the tags. This doesn't necessary mean
// that we can't query using tags not specified here. (REVU it shouldn't.)
//
// Intended usecase is for a gart-tool to say:
//
//    index.OpenIndexManager(OpMode.Read).UsingTags(tags...)
//
func (idx *indexManager) UsingTags(tags ...string) (*indexManager, error) {
	for i, tag := range tags {
		if _, ok := idx.tagmaps[tag]; ok {
			continue // already loaded
		}
		tagmap, e := LoadTagmap(tag, false)
		if e != nil {
			return idx, errors.ErrorWithCause(e, "index.UsingTags: tag[%d]:%q", i, tag)
		}
		idx.tagmaps[tag] = tagmap
	}
	return idx, nil
}

// Indexes the object identified by the Oid.
// REVU see gart-add in /1.0/ for refresh on systemics..
func (idx *indexManager) IndexObject(oid *system.Oid, tags ...string) (bool, error) {
	return false, errors.NotImplemented("indexManager.IndexObject")
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
