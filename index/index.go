// Doost!

package index

import (
	"fmt"
	"os"
	"path/filepath"
	//	"strings"
	//	"time"

	"github.com/alphazero/gart/repo"
	"github.com/alphazero/gart/syslib/bitmap"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/systemic"
)

/// index package errors ///////////////////////////////////////////////////////

var (
	ErrIndexInitialized    = errors.Error("index is already initialized")
	ErrIndexNotInitialized = errors.Error("index is not initialized")
	ErrObjectIndexExist    = errors.Error("%s exists", repo.ObjectIndexPath)
	ErrObjectIndexNotExist = errors.Error("%s does not exist", repo.ObjectIndexPath)
	ErrObjectIndexClosed   = errors.Error("object index is closeed")
	ErrObjectExist         = errors.Error("object exists")
	ErrObjectNotExist      = errors.Error("object does not exist")
)

type Error struct {
	Otype system.Otype
	Oid   *system.Oid
	Err   error
}

func IsObjectExistErr(e0 error) bool {
	e, ok := e0.(Error)
	if !ok {
		return false
	}
	return e.Err == ErrObjectExist
}
func IsObjectNotExistErr(e0 error) bool {
	e, ok := e0.(Error)
	if !ok {
		return false
	}
	return e.Err == ErrObjectNotExist
}
func (v Error) Error() string {
	return fmt.Sprintf("%s - %s object %s ", v.Err.Error(), v.Otype, v.Oid.Fingerprint())
}

/// index ops //////////////////////////////////////////////////////////////////

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
	var debug = debug.For("index.Initialize")
	debug.Printf("reinit: %t", reinit)

	_debug0 := func(path, s string) {
		debug.Printf("reinit:%t - verify-file:%q %s", reinit, path, s)
	}

	switch reinit {
	case true:
		//		println(time.Now().UnixNano())
		_debug := func(path string) { _debug0(path, "does not exist") }
		if e := fs.VerifyFile(repo.ObjectIndexPath); e != nil {
			_debug(repo.ObjectIndexPath)
			return ErrIndexNotInitialized
		}
		if e := fs.VerifyDir(repo.IndexTagmapsPath); e != nil {
			_debug(repo.IndexTagmapsPath)
			return ErrIndexNotInitialized
		}
		if e := fs.VerifyDir(repo.IndexCardsPath); e != nil {
			_debug(repo.IndexCardsPath)
			return ErrIndexNotInitialized
		}
		//		println(time.Now().UnixNano())
		debug.Printf("warn - rm -rf %q", repo.IndexPath)
		if e := os.RemoveAll(repo.IndexPath); e != nil {
			return errors.FaultWithCause(e,
				"index.Initialize (reinit:%t) - os.Mkdir(%s)", reinit, repo.IndexPath)
		}
		//		println(time.Now().UnixNano())
		if e := os.Mkdir(repo.IndexPath, repo.DirPerm); e != nil {
			return errors.FaultWithCause(e,
				"index.Initialize (reinit:%t) - os.Mkdir(%s)", reinit, repo.IndexPath)
		}
		//		println(time.Now().UnixNano())
	default:
		_debug := func(path string) { _debug0(path, "exists") }
		if e := fs.VerifyFile(repo.ObjectIndexPath); e == nil {
			_debug(repo.ObjectIndexPath)
			return ErrIndexInitialized
		}
		if e := fs.VerifyDir(repo.IndexTagmapsPath); e == nil {
			_debug(repo.IndexTagmapsPath)
			return ErrIndexInitialized
		}
		if e := fs.VerifyDir(repo.IndexCardsPath); e == nil {
			_debug(repo.IndexCardsPath)
			return ErrIndexInitialized
		}
	}

	var dirs = []string{repo.IndexCardsPath, repo.IndexTagmapsPath}
	for _, dir := range dirs {
		if e := os.Mkdir(dir, repo.DirPerm); e != nil {
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
	IndexText(bool, string, ...string) (Card, bool, error)
	IndexFile(bool, string, ...string) (Card, bool, error)
	Select(spec selectSpec, tags ...string) ([]*system.Oid, error)
	Exec(Query) ([]*system.Oid, error)
	DeleteObject(oid *system.Oid) (bool, error)
	DeleteObjectsByTag(tags ...string) (int, error)
	RemoveTags(oid *system.Oid, tag ...string) ([]string, error)
	Close() error
}

func OpenIndexManager(opMode OpMode) (IndexManager, error) {
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

func (idx *indexManager) Close() error {
	var err = errors.For("indexManager.Close")

	if idx.oidx == nil || idx.tagmaps == nil {
		return err.Bug("invalid state - already closed")
	}

	// invalidate instance regardless of any errors after this point
	defer func() {
		idx.opMode = 0
		idx.oidx = nil
		idx.tagmaps = nil
	}()

	if e := idx.oidx.closeIndex(); e != nil {
		return err.Bug("on oidx.closeIndex - opMode: %s - closed with e:%v", idx.opMode, e)
	}

	// save loaded tagmaps. (may be nop). Any error is a bug.
	for tag, tagmap := range idx.tagmaps {
		// tagmap compresses on save ...
		if _, e := tagmap.save(); e != nil {
			return err.BugWithCause(e, "on tagmap(%s).Save", tag)
		}
	}

	return nil
}

// Preloads the associated Tagmaps for the tags. This doesn't necessary mean
// that we can't query using tags not specified here. (REVU it shouldn't.)
//
// If this is the first time (in system life-time) that the tag is specified,
// the associated tagmap will be created.
func (idx *indexManager) UsingTags(tags ...string) error {
	var err = errors.For("indexManager.UsingTags")

	for i, tag := range tags {
		if _, ok := idx.tagmaps[tag]; ok {
			continue // already loaded
		}
		tagmap, e := loadTagmap(tag, true) // REVU
		if e != nil {
			return err.ErrorWithCause(e, "tag[%d]:%q", i, tag)
		}
		idx.tagmaps[tag] = tagmap
	}
	return nil
}

func (idx *indexManager) StatText(text string) (*system.Oid, error) {
	md := digest.Sum([]byte(text))
	oid, e := system.NewOid(md[:])
	if e != nil {
		panic(errors.Bug("indexManager.StatText: %v", e))
	}
	return oid, nil
}

// Indexes the text object. If object is new it is added with tags specified. If not
// updated tags (if any) are added. See indexObject.
func (idx *indexManager) IndexText(strict bool, text string, tags ...string) (Card, bool, error) {
	var err = errors.For("indexManager.IndexText")

	// text specific
	md := digest.Sum([]byte(text))
	oid, e := system.NewOid(md[:])
	if e != nil {
		panic(err.BugWithCause(e, "unexpected"))
	}

	var isNew bool
	var card Card
	if cardExists(oid) {
		if strict {
			return nil, false, Error{system.Text, oid, ErrObjectExist}
		}
		card, e = LoadCard(oid)
		if e != nil {
			return card, false, e
		}
	} else {
		var e error
		card, e = NewTextCard(oid, text)
		if e != nil {
			return nil, true, err.BugWithCause(e, "unexpected")
		}
		isNew = true
	}
	return card, isNew, idx.updateIndex(card, isNew, tags...)
}

// Indexes the file object. If object is new it is added with tags specified. If not
// updated tags (if any), or the filename (if new) are added. See indexObject.
func (idx *indexManager) IndexFile(strict bool, filename string, tags ...string) (Card, bool, error) {
	var err = errors.For("indexManager.IndexFile")

	if !filepath.IsAbs(filename) {
		return nil, false, err.InvalidArg("filename must be absolute path")
	}
	md, e := digest.SumFile(filename)
	if e != nil {
		return nil, false, err.InvalidArg(e.Error())
	}
	oid, e := system.NewOid(md[:])
	if e != nil {
		panic(err.BugWithCause(e, "unexpected"))
	}

	var isNew bool
	var card Card
	if cardExists(oid) {
		if strict {
			return nil, false, Error{system.Text, oid, ErrObjectExist}
		}
		card, e = LoadCard(oid)
		if e != nil {
			return card, false, e
		}
		fileCard := card.(*fileCard)
		_, e := fileCard.addPath(filename)
		if e != nil {
			return card, false, e
		}
	} else {
		var e error
		card, e = NewFileCard(oid, filename)
		if e != nil {
			return nil, true, err.BugWithCause(e, "unexpected")
		}
		isNew = true
	}
	return card, isNew, idx.updateIndex(card, isNew, tags...)
}

// REVU see gart-add in /1.0/ for refresh on systemics..
func (idx *indexManager) updateIndex(card Card, isNew bool, tags ...string) error {
	var err = errors.For("indexManager.updateIndex")

	var oid = card.Oid()
	if oid == nil {
		return err.InvalidArg("oid is nil")
	}

	// REVU for now it is ok if no tags are defined

	if isNew {
		// TODO systemics need to be added here
		systemics, e := getObjectSystemics(card)
		if e != nil {
			return err.ErrorWithCause(e, "for new object")
		}
		tags = append(tags, systemics...)

		// 		- object type -> create/update relevant tagmap
		//		- day-tag: MMM-dd-yyyy (e.g. MAR-31-2018) tagmap
		//		- only issue is REVU range-encoding: SIMPLE WAY
		//		  is to check if day-tagmap exists (e.g. is this a new day?)
		//		  and then create tagmaps for ALL days since last date by
		//		  CLONING the previous/last day tagmap. This gets us range
		// 		  encoding. (e.g. object on day T0 is on all daymaps for T0->..
		//        and query (all object created before Tn or range (Ta, Tb)
		//        returns that object by ANDing all day maps in that range.
		key, e := idx.oidx.addObject(oid)
		if e != nil {
			return err.ErrorWithCause(e, "for new object")
		}
		if e := card.setKey(key); e != nil {
			return err.Bug("setKey(%d) for new object - %s", key, e)
		}
	}
	tags = card.addTag(tags...)
	_shouldSave := isNew || len(tags) > 0
	if ok, e := card.save(); e != nil {
		return err.Error("card.Save() - %s", e)
	} else if _shouldSave && !ok {
		return err.Bug("card.Save -> false on newCard")
	}

	// TODO update relevant index tagmaps
	// REVU this basically takes the card's tag def view since tags has been reduced
	//		to the set that is -not- in the card. Certainly is more performant, but
	//		if tagmaps are being 'rebuilt' from cards, this will not work. For this
	// 		function -- updateIndex -- it is OK. For recovery tool, it is not.
	var key = card.Key()
	for _, tag := range tags {
		tagmap, e := idx.loadTagmap(tag, true, true)
		if e != nil {
			return e
		}
		updated := tagmap.update(setBits, uint(key)) // REVU should we change tagmap?
		if updated {
			debug.Printf("updated tagmap (%s) for object (key:%d)", tag, key)
		}
	}
	return nil
}

func (idx *indexManager) loadTagmap(tag string, create, add bool) (*Tagmap, error) {
	var err = errors.For("indexManager.loadTagmap")
	tagmap, ok := idx.tagmaps[tag]
	if !ok {
		var e error
		tagmap, e = loadTagmap(tag, true)
		if e != nil {
			return nil, err.Bug("on loadTagmap(%s) - %v", tag, e)
		}
		if add {
			idx.tagmaps[tag] = tagmap // add it - saved on indexManager.close
		}
	}
	return tagmap, nil
}

// REVU should return TODO ResultSet<T>
func (idx *indexManager) Exec(query Query) ([]*system.Oid, error) {
	var err = errors.For("indexManager.Exec")
	var debug = debug.For("indexManager.Exec")

	debug.Printf("called - query: %v", query)

	return nil, err.NotImplemented()
}

// Select returns the Oids of all objects that have been tagged with all of the
// provided tags. The returned array may be empty but never nil.
// TODO update comments
//
// The indexManager must have been openned in Write op mode and len(tags) must be > 0.
//
// Return nil, error in case of any errors.
func (idx *indexManager) Select(spec selectSpec, tags ...string) ([]*system.Oid, error) {
	var err = errors.For("indexManager.Select")

	if e := spec.verify(); e != nil {
		return nil, e
	}
	if len(tags) == 0 {
		return nil, err.InvalidArg("tags is zero-len")
	}
	var bitmaps []*bitmap.Wahl
	for _, tag := range tags {
		tagmap, ok := idx.tagmaps[tag]
		if !ok {
			var e error
			tagmap, e = loadTagmap(tag, false) // do not create if tag is missing
			if e != nil && e == ErrTagNotExist {
				if spec == All { // we're done here for All
					return []*system.Oid{}, nil
				}
				continue
			} else if e != nil {
				return nil, err.Bug("loadTagmap(%s) - %v", tag, e)
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
	if len(keys) == 0 {
		return []*system.Oid{}, nil
	}
	oids, e := queryFn(keys...)
	if e != nil {
		return nil, e
	}

	// XXX debug only
	debug.Printf("query {select %d for tags %v}", spec, tags)
	debug.Printf("\tkeys (cnt:%d):", len(keys))
	for _, key := range keys {
		debug.Printf("\tkey: %d", key)
	}
	debug.Printf("\toids (cnt:%d):", len(oids))

	for _, oid := range oids {
		debug.Printf("\toid: %s", oid.Fingerprint())
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
	var err = errors.For("indexManager.DeleteObject")

	if idx.opMode != Write {
		return false, err.Error("invalid op mode: %s", idx.opMode)
	}
	if !cardExists(oid) {
		return false, err.Error("does not exist - oid:%s", oid.Fingerprint())
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
			return false, err.Error("card is locked")
		}
		return false, nil
	}
	if ok, e := card.save(); e != nil {
		return false, err.ErrorWithCause(e, "card.save: oid:%s", oid.Fingerprint())
	} else if !ok {
		return false, err.BugWithCause(e, "card.save: oid:%s", oid.Fingerprint())
	}

	// TODO clear the tagmaps of card ..

	return true, nil
}

func (idx *indexManager) DeleteObjectsByTag(tags ...string) (int, error) {
	var err = errors.For("indexManager.DeleteObjectByTag")

	if idx.opMode != Write {
		return 0, err.Bug("invalid op mode: %s", idx.opMode)
	}
	oids, e := idx.Select(All, tags...)
	if e != nil {
		return 0, err.ErrorWithCause(e, "on select(All, ...)")
	}
	// none selected
	if len(oids) == 0 {
		return 0, nil
	}
	// delete selected
	var n int
	for _, oid := range oids {
		if ok, e := idx.DeleteObject(oid); e != nil {
			return n, errors.ErrorWithCause(e, "on DeleteObject")
		} else if ok {
			n++
		}
	}
	return n, nil
}

// RemoveTag removes the specified tags from the object identified by the oid.
//
// Returns []string, nil if successful. The array is set of removed tags.
// Returns []string{}, nil if object was not tagged with any of the specified tag.
// Returns nil, error if object does not exist; is locked; or is marked deleted.
func (idx *indexManager) RemoveTags(oid *system.Oid, tags ...string) ([]string, error) {
	var err = errors.For("indexManager.RemoveTags")

	if idx.opMode != Write {
		return nil, err.Bug("invalid op mode: %s", idx.opMode)
	}
	if !cardExists(oid) {
		return nil, err.Error("does not exist - oid:%s", oid.Fingerprint())
	}

	card, e := LoadCard(oid)
	if e != nil {
		return nil, e
	}
	if card.IsDeleted() {
		return nil, err.Error("card is deleted")
	}
	if card.IsLocked() {
		return nil, err.Error("card is locked")
	}

	/// remove the tag //////////////////////////////////////////////

	updates := card.removeTag(tags...)
	if len(updates) == 0 {
		return updates, nil // exit early - no tagmaps to update
	}

	if ok, e := card.save(); e != nil {
		return nil, err.ErrorWithCause(e, "card.save: oid:%s", oid.Fingerprint())
	} else if !ok {
		return nil, err.BugWithCause(e, "card.save: oid:%s", oid.Fingerprint())
	}

	/// update the tagmap ///////////////////////////////////////////

	for _, tag := range tags {
		tagmap, e := idx.loadTagmap(tag, false, true)
		if e != nil {
			panic(err.BugWithCause(e, "idx.loadTamp (%q)", tag))
		}
		ok := tagmap.update(clearBits, uint(card.Key())) // REVU should we change tagmap?
		if !ok {
			panic(err.Bug("tagmap(%s) update returned false (key:%d)", tag, card.Key()))
		}
	}

	return updates, nil
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

/// systemics //////////////////////////////////////////////////////////////////

// REVU systemics need to be pulled up to system :)
//      top-level commands have bare flag values for e.g. ext=pdf
//		and that needs to be translated to "systemic:ext:pdf".

func getObjectSystemics(card Card) ([]string, error) {
	var err = errors.For("index.getObjectSystemics")

	// day tag
	var systemics = []string{
		systemic.TodayTag(),
		systemic.TypeTag(card.Type().String()),
	}

	// File extension
	// Note: it is possible that a user may choose to define a tag that collides
	// with an, e.g. '.txt.', extension. For now the 'ext:' prefix addresses such
	// a case, but the query api ~is:
	//
	// 		gart-find --ext pdf --tags "...." # find all pdf objects + tags
	//  or
	// 		gart-find --no-ext --tags "...."  # find all objects with no extension + ..
	//
	// so even if user (for whatever reason) has applied e.g. '.txt' tag, it can
	// not collide in the tag.Map. Of course, the prefix is necessary.
	//
	// ex: "ext:pdf" # .pdf extension
	// ex: "ext:"    # no extension
	if card.Type() == system.File {
		fd, e := fs.GetFileDetails(card.(FileCard).Paths()[0])
		if e != nil {
			return nil, err.ErrorWithCause(e, "using card.path[0]")
		}
		var ext = "systemic:ext:"
		if fd.Ext != "" {
			ext += fd.Ext[1:]
		}
		systemics = append(systemics, systemic.ExtTag(ext))
	}

	return systemics, nil
}

/*
func typeTag(otype system.Otype) string {
	return fmt.Sprintf("systemic:type:%s", otype.String())
}

// All gart objects are tagged with the journal date. This function retuns
// a tag name of form "MMM-dd-YYYY" (e.g. MAR-21-2018).
func dayTag() string {
	y, m, d := time.Now().Date()
	return fmt.Sprintf("systemic:day:%s-%02d-%d", strings.ToLower(m.String()[:3]), d, y)
}
*/
