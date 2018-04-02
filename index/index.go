// Doost!

package index

import (
	"fmt"
	"os"

	"github.com/alphazero/gart/syslib/digest"
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

	system.Debugf("index.Initialize: reinit: %t", reinit)

	_debug0 := func(path, s string) {
		system.Debugf("index.Initialize(%t): verify-file:%q %s", reinit, path, s)
	}

	switch reinit {
	case true:
		_debug := func(path string) { _debug0(path, "does not exist") }
		if e := fs.VerifyFile(system.ObjectIndexPath); e != nil {
			_debug(system.ObjectIndexPath)
			return ErrIndexNotInitialized
		}
		if e := fs.VerifyDir(system.IndexTagmapsPath); e != nil {
			_debug(system.IndexTagmapsPath)
			return ErrIndexNotInitialized
		}
		if e := fs.VerifyDir(system.IndexCardsPath); e != nil {
			_debug(system.IndexCardsPath)
			return ErrIndexNotInitialized
		}
		system.Debugf("warn - rm -rf %q", system.IndexPath)
		if e := os.RemoveAll(system.IndexPath); e != nil {
			return errors.FaultWithCause(e,
				"index.Initialize (reinit:%t) - os.Mkdir(%s)", reinit, system.IndexPath)
		}
		if e := os.Mkdir(system.IndexPath, system.DirPerm); e != nil {
			return errors.FaultWithCause(e,
				"index.Initialize (reinit:%t) - os.Mkdir(%s)", reinit, system.IndexPath)
		}
	default:
		_debug := func(path string) { _debug0(path, "exists") }
		if e := fs.VerifyFile(system.ObjectIndexPath); e == nil {
			_debug(system.ObjectIndexPath)
			return ErrIndexInitialized
		}
		if e := fs.VerifyDir(system.IndexTagmapsPath); e == nil {
			_debug(system.IndexTagmapsPath)
			return ErrIndexInitialized
		}
		if e := fs.VerifyDir(system.IndexCardsPath); e == nil {
			_debug(system.IndexCardsPath)
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
	if system.Debug {
		oidx.Print(os.Stderr)
	}

	var idxmgr = &indexManager{
		opMode:  opMode,
		oidx:    oidx,
		tagmaps: make(map[string]*Tagmap),
	}

	system.Debugf("index.OpenIndexManager: opMode: %s - openned.", opMode)
	return idxmgr, nil
}

func (idx *indexManager) Close() error {

	if idx.oidx == nil || idx.tagmaps == nil {
		return errors.Bug("indexManager.Close: invalid state - already closed")
	}

	if system.Debug {
		idx.oidx.Print(os.Stderr)
	}
	var e = idx.oidx.closeIndex()

	system.Debugf("index.OpenIndexManager: opMode: %s - closed (e: %s).", idx.opMode, e)
	// invalidate instance regardless of any errors
	idx.opMode = 0
	idx.oidx = nil
	idx.tagmaps = nil

	return e
}

// Preloads the associated Tagmaps for the tags. This doesn't necessary mean
// that we can't query using tags not specified here. (REVU it shouldn't.)
//
// If this is the first time (in system life-time) that the tag is specified,
// the associated tagmap will be created.
func (idx *indexManager) UsingTags(tags ...string) error {
	debug := func(s string) { system.Debugf("index.OpenIndexManager: " + s) }

	for i, tag := range tags {
		if _, ok := idx.tagmaps[tag]; ok {
			debug(tag + " is already loaded")
			continue // already loaded
		}
		tagmap, e := loadTagmap(tag, true) // REVU
		if e != nil {
			return errors.ErrorWithCause(e, "index.UsingTags: tag[%d]:%q", i, tag)
		}
		debug(tag + " loaded")
		idx.tagmaps[tag] = tagmap
	}
	return nil
}

// Indexes the object identified by the Oid & system.Otype
func (idx *indexManager) IndexText(text string, tags ...string) (int64, bool, error) {
	md := digest.Sum([]byte(text))
	oid, e := system.NewOid(md[:])
	if e != nil {
		panic(errors.BugWithCause(e, "indexManager.IndexText: unexpected"))
	}

	var card Card
	var isNew bool
	if !cardExists(oid) {
		var e error
		card, e = NewTextCard(oid, text)
		if e != nil {
			return -1, true, errors.Bug("indexManager.IndexText: - %s", e)
		}
	} else {
		card, e = loadCard(oid)
	}

	key, e := idx.indexObject(card, isNew, tags...)
	return key, isNew, e
}

func (idx *indexManager) IndexFile(filename string, tags ...string) (int64, bool, error) {
	return -1, false, errors.NotImplemented("indexManager.IndexFile")
}

// REVU see gart-add in /1.0/ for refresh on systemics..
func (idx *indexManager) indexObject(card Card, isNew bool, tags ...string) (int64, error) {

	var oid = card.Oid()
	if oid == nil {
		return card.Key(), errors.InvalidArg("indexManager.indexObject", "oid", "nil")
	}

	// REVU for now it is ok if no tags are defined
	// TODO systemics need to be added here as well

	if isNew {
		key, e := idx.oidx.addObject(oid)
		if e != nil {
			return key, errors.ErrorWithCause(e, "IndexManager.indexObject")
		}
		if e := card.setKey(key); e != nil {
			return key, errors.Bug("indexManager.indexObject: setKey(%d) - %s", key, e)
		}
		if ok, e := card.save(); e != nil {
			return key, errors.Error("indexManager.indexObject: card.Save() - %s", e)
		} else if !ok {
			return key, errors.Bug("indexManager.indexObject: card.Save -> false on newCard")
		}
	}

	// update all tagmaps for card.Key
	var key = card.Key()
	for _, tag := range tags {
		system.Debugf("TODO - set tagmap bit %d for tag %s ", key, tag)
	}

	return key, nil
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

	system.Debugf("indexManager.SelectObjects: selected key ids:\n")
	for _, key := range resmap.Bits() {
		fmt.Printf("key: %d\n", key)
	}

	// TODO pass keys to objects-index to get the OID list
	// oids = GetObjectIds(resmap.Bits())

	return oids, nil
}
