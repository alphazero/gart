// Doost!

package index

import (
	"fmt"
	"os"
	"path/filepath"

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
type IndexManager interface {
	UsingTags(tags ...string) error
	IndexText(bool, string, ...string) (Card, bool, error)
	IndexFile(bool, string, ...string) (Card, bool, error)
	Select(spec selectSpec, tags ...string) ([]*system.Oid, error)
	Search(Query) ([]*system.Oid, error) // REVU this is really Search
	DeleteObject(oid *system.Oid) (bool, error)
	DeleteObjectsByTag(tags ...string) (int, error)
	RemoveTags(oid *system.Oid, tag ...string) ([]string, error)

	Rollback() error
	Close(commit bool) error
}

type indexManager struct {
	opMode  OpMode
	oidx    *oidxFile
	tagmaps map[string]*Tagmap
	cards   map[string]Card
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
		cards:   make(map[string]Card),
	}

	return idxmgr, nil
}

// REVU calling rollback now will leave object.idx in an inconsistent state
func (idx *indexManager) Rollback() error {
	var err = errors.For("indexManager.Rollback")
	var debug = debug.For("indexManager.Rollback")
	debug.Printf("modified cards:%d tagmaps:%d", len(idx.cards), len(idx.tagmaps))

	for key, _ := range idx.cards {
		//		if e := card.removeWip(); e != nil {
		//			return err.Bug("on card[%s].removeWip - e:%v", card.Oid().Fingerprint(), e)
		//		}
		delete(idx.cards, key)
	}
	for key, _ := range idx.tagmaps {
		delete(idx.tagmaps, key)
	}

	if e := idx.oidx.closeIndex(false); e != nil {
		return err.ErrorWithCause(e, "on Rollback")
	}

	return nil
}

func (idx *indexManager) Close(commit bool) error {
	var err = errors.For("indexManager.Close")
	var debug = debug.For("indexManager.Close")
	debug.Printf("called - commit:%t idx.opMode:%s", commit, idx.opMode)

	if idx.oidx == nil || idx.tagmaps == nil {
		return err.Bug("invalid state - already closed")
	}

	var idxModified bool = (len(idx.tagmaps) + len(idx.cards)) > 0

	switch idx.opMode {
	case Write, Compact:
		if idxModified && !commit {
			return err.Error("uncommitted transactions")
		}
	default:
		debug.Printf("read-only mode - no transactions to commit")
	}

	// invalidate instance regardless of any errors after this point
	// using the index manager after this call returns will panic due to nils
	defer func() {
		idx.opMode = 0
		idx.oidx = nil
		idx.tagmaps = nil
		idx.cards = nil
	}()

	// note: close must be called for object.idx file.
	if e := idx.oidx.closeIndex(commit); e != nil {
		return err.Bug("on oidx.closeIndex - opMode: %s - closed with e:%v", idx.opMode, e)
	}

	if !commit {
		return nil
	}

	// note: cards are in-memory objects
	// note: save is a nop if card is not modified
	for oid, card := range idx.cards {
		debug.Printf("saving wip card - %s", oid)
		if ok, e := card.saveWip(); !ok || e != nil {
			return err.Bug("card.saveWip returned oid:%s ok:%t e:%v", oid, ok, e)
		}
		if ok, e := card.save(); e != nil || !ok {
			return err.Bug("on card[%s].save - ok:%t e:%v", oid, ok, e)
		}
		debug.Printf("saved card[%s]", oid)
	}

	// note: tagmaps are in-memory objects
	// note: save is a nop if tagmap is not modified
	for tag, tagmap := range idx.tagmaps {
		// tagmap compresses on save so no need to compress it
		if ok, e := tagmap.save(); e != nil {
			return err.BugWithCause(e, "on tagmap(%s).Save", tag)
		} else if !ok {
			return err.Bug("tagmap[%s].save returned false, nil", tag)
		}
		debug.Printf("saved tagmap[%s]", tag)
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
	var debug = debug.For("indexManager.IndexFile")
	debug.Printf("strict:%t filename:%q", strict, filename)

	if !filepath.IsAbs(filename) {
		return nil, false, err.InvalidArg("filename must be absolute path")
	}
	md, e := digest.SumFile(filename)
	if e != nil {
		// REVU don't wrap the error as it is 99% os.ErrNotExist due to funky path issues.
		//      Problem seems to be a Golang bug with embedded \n in file name. filepath
		//      accepts it, (OS X) ls and cat accept it, but SumFile chokes on it.
		//
		//      So return the original error so the top level function can check for it and
		//      not stop a session because of this issue.
		return nil, false, e
	}
	oid, e := system.NewOid(md[:])
	if e != nil {
		panic(err.BugWithCause(e, "unexpected"))
	}

	var isNew bool
	var card Card = idx.cards[oid.String()]
	switch {
	case cardExists(oid):
		// REVU strict is strictly broken
		//      idea behind strict is 'don't index an existing card'. but
		//      what if in the same session multiple files are resolving to the same
		//      object? In that case the second case statement (card != nil) will select
		//      and paths will be updated.
		//
		//      And if we include that as well for strict filtering, then how to add paths
		//      explicitly?
		if strict {
			return nil, false, Error{system.Text, oid, ErrObjectExist}
		}
		card, e = LoadCard(oid)
		if e != nil {
			return card, false, e
		}
		fallthrough
		//		fileCard := card.(*fileCard)
		//		if _, e := fileCard.addPath(filename); e != nil {
		//			return card, false, e
		//		}
	case card != nil:
		fileCard := card.(*fileCard)
		if _, e := fileCard.addPath(filename); e != nil {
			return card, false, e
		}
	default:
		var e error
		card, e = NewFileCard(oid, filename)
		if e != nil {
			return nil, true, err.BugWithCause(e, "unexpected")
		}
		isNew = true
	}
	return card, isNew, idx.updateIndex(card, isNew, tags...)
}

func (idx *indexManager) updateIndex(card Card, isNew bool, tags ...string) error {
	var err = errors.For("indexManager.updateIndex")
	var debug = debug.For("indexManager.updateIndex")

	var oid = card.Oid()
	if oid == nil {
		return err.InvalidArg("oid is nil")
	}

	if isNew {
		// TODO day-tag: MMM-dd-yyyy range-encoding
		systemics, e := getObjectSystemics(card)
		if e != nil {
			return err.ErrorWithCause(e, "for new object")
		}
		tags = append(tags, systemics...)

		key, e := idx.oidx.addObject(oid)
		if e != nil {
			return err.ErrorWithCause(e, "for new object")
		}
		if e := card.setKey(key); e != nil {
			return err.Bug("setKey(%d) for new object - %s", key, e)
		}
	}

	tags = card.addTag(tags...)
	if idx.cards[oid.String()] == nil && (card.IsNew() || card.isModified()) {
		//		var oidfp = oid.Fingerprint()
		//		debug.Printf("saving wip card - %s", oidfp)
		//		if ok, e := card.saveWip(); !ok || e != nil {
		//			return err.Bug("card.saveWip returned oid:%s ok:%t e:%v", oid.Fingerprint(), ok, e)
		//		}
		debug.Printf("add card %s to cards map", oid.Fingerprint())
		idx.cards[oid.String()] = card
	}

	var key = card.Key()
	for _, tag := range tags {
		debug.Printf("load tagmap %q", tag)
		tagmap, e := idx.loadTagmap(tag, true, true)
		if e != nil {
			return e
		}
		updated := tagmap.update(setBits, uint(key))
		if updated { // XXX debug
			debug.Printf("updated tagmap (%s) for object (key:%d)", tag, key)
		} // XXX END
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
func (idx *indexManager) Search(qx Query) ([]*system.Oid, error) {
	var err = errors.For("indexManager.Search")
	var debug = debug.For("indexManager.Search")

	var q = qx.asQuery()

	if len(q.include) == 0 {
		q.IncludeTags(systemic.GartTag())
		// REVU we can just jump ahead to result here.
	}
	debug.Printf("called q: %v", q)

	var e error

	var exmap = bitmap.NewWahl()
	var excluded []*bitmap.Wahl
	for tag, _ := range q.exclude {
		tagmap, ok := idx.tagmaps[tag]
		if !ok {
			if tagmap, e = loadTagmap(tag, false); e != nil && e == ErrTagNotExist {
				continue
			}
		}
		excluded = append(excluded, tagmap.bitmap)
	}
	if exmap, e = bitmap.Or(excluded...); e != nil {
		return nil, err.ErrorWithCause(e, "on exluded set OR")
	}

	// the included tags
	var inmap = bitmap.NewWahl() // empty set
	var included []*bitmap.Wahl
	for tag, _ := range q.include {
		tagmap, ok := idx.tagmaps[tag]
		if !ok {
			if tagmap, e = loadTagmap(tag, false); e != nil && e == ErrTagNotExist {
				return []*system.Oid{}, nil // fast-path return w/ empty set
			} else if e != nil {
				return nil, err.Bug("loadTagmap(%s) - %v", tag, e)
			}
		}
		included = append(included, tagmap.bitmap)
	}
	if inmap, e = bitmap.And(included...); e != nil {
		return nil, err.ErrorWithCause(e, "on included set AND")
	}

	if inmap.Len() == 0 {
		return []*system.Oid{}, nil
	}

	if exmap.Len() > 0 {
		if exmap, e = inmap.Xor(exmap); e != nil {
			return nil, err.ErrorWithCause(e, "on XOR")
		}
		if inmap, e = inmap.And(exmap); e != nil {
			return nil, err.ErrorWithCause(e, "on AND")
		}
	}
	bits := inmap.Bits()
	return idx.oidx.getOids([]int(bits)...)
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
	resmap, e := bitmap.And(bitmaps...)
	return []int(resmap.Bits()), e
}

// Returns the logical OR of the following bitmaps.
func (idx *indexManager) bitmapsOR(bitmaps []*bitmap.Wahl) ([]int, error) {
	resmap, e := bitmap.And(bitmaps...)
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

	var systemics = []string{
		systemic.GartTag(), // used for all inclusive mapping
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
		var ext string // = "systemic:ext:"
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
