// Doost!

package index

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system" // TODO  REVU decision for OID in system ..
)

/// index package errors ///////////////////////////////////////////////////////

var (
	ErrIndexInitialized    = errors.Error("index is already initialized")
	ErrIndexNotInitialized = errors.Error("index is not initialized")
	ErrObjectIndexExist    = errors.Error("%s exists", system.ObjectIndexPath)
	ErrObjectIndexNotExist = errors.Error("%s does not exist", system.ObjectIndexPath)
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

// index.Initialize creates the canonical index files for the repo. The index
// directory itself is assumed to exist per toplevel gart initialization.
//
// If reinit is false & the index is already intialized, ErrIndexInitialized
// is returned.
//
// If reinit is true * the index is not initialized, ErrIndexNotInitialized
// is returned.
//
// Function may also return an error with cause.
func Initialize(reinit bool) error {

	system.Debugf("index.Initialize: reinit:%t", reinit)

	switch reinit {
	case true:
		if e := fs.VerifyFile(system.ObjectIndexPath); e != nil {
			return ErrIndexNotInitialized
		}
		if e := fs.VerifyDir(system.IndexTagmapsPath); e != nil {
			return ErrIndexNotInitialized
		}
		if e := fs.VerifyDir(system.IndexCardsPath); e != nil {
			return ErrIndexNotInitialized
		}
		// REVU this is dangerous - a bug in system.runtime.go had 'IndexPath' as
		// USER-HOME << !!! as /Users/alphazero/index - TODO think about this.
		if e := os.RemoveAll(system.IndexPath); e != nil {
			return errors.FaultWithCause(e,
				"index.Initialize (reinit:%t) - os.Mkdir(%s)", reinit, system.IndexPath)
		}
		if e := os.Mkdir(system.IndexPath, system.DirPerm); e != nil {
			return errors.FaultWithCause(e,
				"index.Initialize (reinit:%t) - os.Mkdir(%s)", reinit, system.IndexPath)
		}
	default:
		if e := fs.VerifyFile(system.ObjectIndexPath); e == nil {
			system.Debugf("index.Initialize: verify-file:%q not exists", system.ObjectIndexPath)
			return ErrIndexInitialized
		}
		if e := fs.VerifyDir(system.IndexTagmapsPath); e == nil {
			system.Debugf("index.Initialize: verify-dir:%q not exists", system.IndexTagmapsPath)
			return ErrIndexInitialized
		}
		if e := fs.VerifyDir(system.IndexCardsPath); e == nil {
			system.Debugf("index.Initialize: verify-dir:%q not exists", system.IndexCardsPath)
			return ErrIndexInitialized
		}
	}

	var dirs = []string{system.IndexCardsPath, system.IndexTagmapsPath}
	for _, dir := range dirs {
		if e := os.Mkdir(dir, system.DirPerm); e != nil {
			return errors.FaultWithCause(e,
				"index.Initialize (reinit:%t) - os.Mkdir(%s)", reinit, dir)
		}
	}

	if e := createObjectIndex(); e != nil {
		return errors.ErrorWithCause(e,
			"index.InitializeRepo: error creating objects index")
	}

	return nil
}

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
		tagmap, e := loadTagmap(tag, false)
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

	tagmap0, e := loadTagmap(tags[0], false)
	if e != nil {
		return nil, e
	}

	var oids = []*system.Oid{} // initial empty result set

	var resmap = tagmap0.bitmap
	for _, tag := range tags[1:] {
		tagmap, e := loadTagmap(tag, false)
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
