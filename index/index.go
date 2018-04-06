// Doost!

package index

import (
	"os"
	"path/filepath"

	"github.com/alphazero/gart/syslib/bitmap"
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

type IndexManager interface {
	UsingTags(tags ...string) error
	IndexText(text string, tags ...string) (Card, bool, error)
	IndexFile(filename string, tags ...string) (Card, bool, error)
	Select(spec selectSpec, tags ...string) ([]*system.Oid, error)
	DeleteObject(oid *system.Oid) (bool, error)
	DeleteObjectsByTag(tags ...string) (int, error)
	Close() error
}

func OpenIndexManager(opMode OpMode) (*indexManager, error) {
	oidx, e := openObjectIndex(opMode)
	if e != nil {
		return nil, e
	}
	if system.Debug {
		system.Debugf("index.OpenIndexManager: oidx on Open ===============")
		oidx.Print(os.Stderr)
		system.Debugf("============================================ end ===\n")
	}

	var idxmgr = &indexManager{
		opMode:  opMode,
		oidx:    oidx,
		tagmaps: make(map[string]*Tagmap),
	}

	system.Debugf("index.OpenIndexManager: opMode: %s - openned", opMode)
	return idxmgr, nil
}

func (idx *indexManager) Close() error {

	if idx.oidx == nil || idx.tagmaps == nil {
		return errors.Bug("indexManager.Close: invalid state - already closed")
	}

	if system.Debug {
		system.Debugf("indexManager.Close: -- oidx on Close ===============")
		idx.oidx.Print(os.Stderr)
		system.Debugf("============================================ end ===\n")
	}
	var e = idx.oidx.closeIndex()

	system.Debugf("indexManager.Close: opMode: %s - closed with e:%v", idx.opMode, e)

	// save loaded tagmaps. (may be nop). Any error is a bug.
	for tag, tagmap := range idx.tagmaps {
		if ok, e := tagmap.save(); e != nil {
			panic(errors.BugWithCause(e, "indexManager.Close: on tagmap(%s).Save", tag))
		} else if ok {
			system.Debugf("updated tagmap %s", tag)
		}
	}

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

// Indexes the text object. If object is new it is added with tags specified. If not
// updated tags (if any) are added. See indexObject.
func (idx *indexManager) IndexText(text string, tags ...string) (Card, bool, error) {
	md := digest.Sum([]byte(text))
	oid, e := system.NewOid(md[:])
	if e != nil {
		panic(errors.BugWithCause(e, "indexManager.IndexText: unexpected"))
	}

	var card Card
	var isNew = !cardExists(oid)
	if isNew {
		var e error
		card, e = NewTextCard(oid, text)
		if e != nil {
			return nil, true, errors.Bug("indexManager.IndexText: - %s", e)
		}
	} else {
		card, e = LoadCard(oid)
		if e != nil {
			return card, false, e
		}
	}

	return card, isNew, idx.updateIndex(card, isNew, tags...)
}

// Indexes the file object. If object is new it is added with tags specified. If not
// updated tags (if any), or the filename (if new) are added. See indexObject.
func (idx *indexManager) IndexFile(filename string, tags ...string) (Card, bool, error) {
	if !filepath.IsAbs(filename) {
		return nil, false, errors.InvalidArg("indexManager.IndexFile", "filename", "not absolute")
	}
	md, e := digest.SumFile(filename)
	if e != nil {
		panic(errors.BugWithCause(e, "indexManager.IndexFile: unexpected"))
	}
	oid, e := system.NewOid(md[:])
	if e != nil {
		panic(errors.BugWithCause(e, "indexManager.IndexFile: unexpected"))
	}
	var card Card
	var isNew = !cardExists(oid)
	if isNew {
		var e error
		card, e = NewFileCard(oid, filename)
		if e != nil {
			return card, true, errors.Bug("indexManager.IndexFile: - %s", e)
		}
	} else {
		card, e = LoadCard(oid)
		if e != nil {
			return card, false, e
		}
		fileCard := card.(*fileCard)
		ok, e := fileCard.addPath(filename)
		if e != nil {
			return card, false, e
		}
		if ok {
			system.Debugf("indexManager.IndexFile: added path: %q", filename)
		}
	}

	return card, isNew, idx.updateIndex(card, isNew, tags...)
}

// REVU see gart-add in /1.0/ for refresh on systemics..
func (idx *indexManager) updateIndex(card Card, isNew bool, tags ...string) error {

	var oid = card.Oid()
	if oid == nil {
		return errors.InvalidArg("indexManager.indexObject", "oid", "nil")
	}

	// REVU for now it is ok if no tags are defined
	// TODO systemics need to be added here as well

	//	var updates []string // new tags
	if isNew {
		key, e := idx.oidx.addObject(oid)
		if e != nil {
			return errors.ErrorWithCause(e, "IndexManager.indexObject")
		}
		if e := card.setKey(key); e != nil {
			return errors.Bug("indexManager.indexObject: setKey(%d) - %s", key, e)
		}
	}
	tags = card.addTag(tags...)
	system.Debugf("indexManager.indexObject: card. modified:%t", card.isModified())
	_shouldSave := isNew || len(tags) > 0
	system.Debugf("indexManager.indexObject: saving card")
	if ok, e := card.save(); e != nil {
		return errors.Error("indexManager.indexObject: card.Save() - %s", e)
	} else if _shouldSave && !ok {
		return errors.Bug("indexManager.indexObject: card.Save -> false on newCard")
	}

	// TODO update relevant index tagmaps
	// REVU this basically takes the card's tag def view since tags has been reduced
	//		to the set that is -not- in the card. Certainly is more performant, but
	//		if tagmaps are being 'rebuilt' from cards, this will not work. For this
	// 		function -- updateIndex -- it is OK. For recovery tool, it is not.
	var key = card.Key()
	for _, tag := range tags {
		tagmap, ok := idx.tagmaps[tag]
		if !ok {
			var e error
			tagmap, e = loadTagmap(tag, true)
			if e != nil {
				return errors.Bug("indexManager.updateIndex: loadTagmap(%s) - %v",
					tag, e)
			}
			idx.tagmaps[tag] = tagmap // add it - saved on indexManager.close
		}
		updated := tagmap.update(uint(key)) // REVU should we change tagmap?
		if updated {
			system.Debugf("updated tagmap (%s) for object (key:%d)", tag, key)
		}
	}

	return nil
}

// Select returns the Oids of all objects that have been tagged with all of the
// provided tags. The returned array may be empty but never nil. If one or more of
// the tags are undefined, the empty set with no error is returned.
//
// The indexManager must have been openned in Write op mode and len(tags) must be > 0.
//
// Return nil, error in case of any errors.
func (idx *indexManager) Select(spec selectSpec, tags ...string) ([]*system.Oid, error) {

	if e := spec.verify(); e != nil {
		return nil, errors.ErrorWithCause(e, "indexManager.Select")
	}
	if len(tags) == 0 {
		return nil, errors.InvalidArg("indexManager.Select", "len(tags)", "0")
	}
	var bitmaps []*bitmap.Wahl
	for _, tag := range tags {
		tagmap, ok := idx.tagmaps[tag]
		if !ok {
			var e error
			tagmap, e = loadTagmap(tag, false) // do not create if tag is missing
			if e != nil && e == ErrTagNotExist {
				if spec == All { // we're done here for All
					return nil, nil
				}
				continue
			} else if e != nil {
				return nil, errors.Bug("indexManager.select: loadTagmap(%s) - %v", tag, e)
			}
		}
		bitmaps = append(bitmaps, tagmap.bitmap)
	}

	var selectFn func([]*bitmap.Wahl) ([]int, error)
	var queryFn func(...int) ([]*system.Oid, error)
	switch spec {
	case All:
		selectFn = idx.bitmapsAND
		queryFn = idx.oidx.getOids
	case Any:
		selectFn = idx.bitmapsOR
		queryFn = idx.oidx.getOids
	case None:
		selectFn = idx.bitmapsOR
		queryFn = idx.oidx.getOidsExcluding
	}
	keys, e := selectFn(bitmaps)
	if e != nil {
		return nil, e
	}
	oids, e := queryFn(keys...)
	if e != nil {
		return nil, e
	}

	// XXX debug only
	system.Debugf("query {select %d for tags %v}", spec, tags)
	system.Debugf("\tkeys (cnt:%d):", len(keys))
	for _, key := range keys {
		system.Debugf("\tkey: %d", key)
	}
	system.Debugf("\toids (cnt:%d):", len(oids))

	for _, oid := range oids {
		system.Debugf("\toid: %s", oid.Fingerprint())
	}
	// XXX debug only - END

	return oids, nil
}

// Returns the logical AND of the following bitmaps.
func (idx *indexManager) bitmapsAND(bitmaps []*bitmap.Wahl) ([]int, error) {
	resmap, e := bitmap.AND(bitmaps...)
	return []int(resmap.Bits()), e
}

// Returns the logical OR of the following bitmaps.
func (idx *indexManager) bitmapsOR(bitmaps []*bitmap.Wahl) ([]int, error) {
	resmap, e := bitmap.AND(bitmaps...)
	return []int(resmap.Bits()), e
}

func (idx *indexManager) DeleteObject(oid *system.Oid) (bool, error) {
	if !cardExists(oid) {
		return false, errors.Error(
			"indexManager.DeleteObject: does not exist - oid:%s", oid.Fingerprint())
	}
	card, e := LoadCard(oid)
	if e != nil {
		return false, e
	}
	if card.IsDeleted() {
		return false, nil
	}

	if ok := card.markDeleted(); !ok {
		if card.IsLocked() {
			return false, errors.Error("indexManager.DeleteObject: card is locked")
		}
		return false, errors.Bug("indexManager.DeleteObject - oid:%s", oid.Fingerprint())
	}
	if ok, e := card.save(); e != nil {
		return false, errors.ErrorWithCause(e, "indexManager.DeleteObject: card.save: oid:%s",
			oid.Fingerprint())
	} else if !ok {
		return false, errors.BugWithCause(e, "indexManager.DeleteObject: card.save: oid:%s",
			oid.Fingerprint())
	}
	return true, nil
}

func (idx *indexManager) DeleteObjectsByTag(tags ...string) (int, error) {
	oids, e := idx.Select(All, tags...)
	if e != nil {
		return 0, errors.ErrorWithCause(e, "indexManager.DeleteObjectsByTag")
	}
	// none selected
	if len(oids) == 0 {
		return 0, nil
	}
	// delete selected
	var n int
	for _, oid := range oids {
		if ok, e := idx.DeleteObject(oid); e != nil {
			return n, errors.ErrorWithCause(e, "indexManager.DeleteObjectsByTag")
		} else if ok {
			n++
		}
	}
	return n, nil
}

/// selectSpec /////////////////////////////////////////////////////////////////

type selectSpec byte

const (
	_ selectSpec = iota
	All
	Any
	None
)

func (v selectSpec) String() string {
	switch v {
	case All:
	case Any:
	case None:
	}
	panic(errors.Bug("selectSpec.String: invalid select spec: %d", v))
}
func (v selectSpec) verify() error {
	switch v {
	case All, Any, None:
		return nil
	}
	return errors.Bug("selectSpec.verify: invalid select spec: %d", v)
}
