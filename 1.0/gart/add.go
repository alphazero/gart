// Doost!

package gart

import (
	"fmt"
	"os"
	"sort"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/tag"
)

type AddOpResult struct {
	opResult
	Card      index.Card
	DupObject bool
	DupPath   bool
}

// Adds the filesystem object (fpath) to the provided gart.Context.
// This function is the top level api for archiving a single object.
//
// Any necessary directories and files (card file & parent dirs, etc.) that are
// not already created will be created. .gart/index/oid-tags.idx will also be
// updated.
//
// If the filesystem object's signature matches an existing index.Card, the
// path will be updated.
//
// On success, function will return the associated index.Card, a flag indicating
// if the object is 'dup', and nil error.
//
// On error, (REVU for now), the index.Card and flag semantics are undefined.
//
// TODO return an AddOpResult
func AddObject(ctx *OpContext, fpath string, tags []string) OpResult {

	var result = &AddOpResult{}

	// file details _____________________

	fds, e := fs.GetFileDetails(fpath)
	if e != nil {
		return result.onError(e, "fs.GetFileDetails")
	}

	// object identity __________________

	oid, e := index.ObjectId(fds.Path)
	if e != nil {
		// if PathError and ErrPermission skip this item but don't abort
		if pe, ok := e.(*os.PathError); ok && pe.Err == os.ErrPermission {
			return result.onError(e, "index.ObjectId - PathError")
		}
		// anything else (?) well, let's treat it as a bug for now.
		return result.onBug(e, "index.ObjectId")
	}

	// object tags ______________________
	// TODO
	// get bitmaps for (user) tags and (file) systemics
	var tbm bitmap.Bitmap // tags bitmap
	var sbm bitmap.Bitmap // systemic bitmap
	// move to index BEGIN
	_, e = updateTagsForFile(ctx.Tagmap, tags, &fds)
	if e != nil {
		return result.onBug(e, "UpdateTagsForFile")
	}
	// move to index END

	// HERE look, if 'index' is in charge of indexing things, then
	//      it should just updated the object index as well. All we
	//      needed to do here was above: convert tags from []string to
	//      bitmaps and set some systemics.
	//
	//      So the only Q here is: do we really return a 4-tuple :) or
	//      possibly hack the Card for Card.IsNew() (which is easy to do
	//      since if a card was read from a file it will say no, otherwise yes.
	//
	//      so then this below becomes
	//
	//    card, updated, e := index.AddOrUpdateObject(home, fpath, tbm, sbm)
	//
	//    'card' is an index card for an object. that's it.
	// object card ______________________
	//
	card, updated, e := index.AddOrUpdateCard(ctx.Home, oid, fpath, tbm, sbm)
	if e != nil {
		return result.onError(e, "index.GetOrCreateCard")
	}
	newCard := card.Revision() == 0
	result.Card = card
	result.DupObject = !newCard
	result.DupPath = len(card.Paths()) > 1

	// object index _____________________
	// updat oid-tags index if new card or if existing oid has updated bitmaps
	// TODO
	if newCard || updated {
	}

	return result
}

// REVU below this should al be in tag ------------

func updateTagsForFile(tagmap tag.Map, tags []string, fds *fs.FileDetails) ([]int, error) {
	// REVU do we need systemics (names) returned here?
	var ids []int
	_, stids, e := addSystemicTags(tagmap, fds)
	if e != nil {
		return ids, e
	}
	utids, e := addTags(tagmap, tags...)
	if e != nil {
		return ids, e
	}

	ids = append(utids, stids...)
	sort.IntSlice(ids).Sort()

	return ids, nil
}

func addSystemicTags(tagmap tag.Map, fds *fs.FileDetails) ([]string, []int, error) {
	systemics := tag.AllSystemic(fds)
	ids, e := addTags(tagmap, systemics...)
	return systemics, ids, e
}

func addTags(tagmap tag.Map, tags ...string) ([]int, error) {

	var ids = make([]int, len(tags))

	for i, name := range tags {
		_, id, e := tagmap.Add(name)
		if e != nil {
			return nil, fmt.Errorf("bug - gart-add: addFileTags: Add %q - %s", name, e)
		}
		if _, _, e := tagmap.IncrRefcnt(name); e != nil {
			return nil, fmt.Errorf("bug - gart-add: addFileTags: IncrRefCnt %q - %s", name, e)
		}
		ids[i] = id
	}
	return ids, nil
}
