// Doost!

package index

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
)

/// Object IDs /////////////////////////////////////////////////////////////////

const (
	OidBytes = 32 // REVU why export this?
)

// export the type but keep internals private to index package
type OID struct {
	dat [OidBytes]byte
}

// REVU: this should be the only way to get an OID
func ObjectId(fpath string) (*OID, error) {

	md, e := digest.SumFile(fpath)
	if e != nil {
		return nil, e
	}

	return newOid(md), nil
}

// internal func panics on errors
func newOid(bytes []byte) *OID {

	if bug := validateOidBytes(bytes); bug != nil {
		panic(fmt.Errorf("bug - index.newOid: invalid arg - %s", bug))
	}
	var oid OID
	copy(oid.dat[:], bytes[:OidBytes])

	return &oid
}

func validateOidBytes(bytes []byte) error {
	if len(bytes) < OidBytes {
		return fmt.Errorf("bug - invalid OID bytes - len: %d", len(bytes))
	}
	for _, b := range bytes {
		if b != 0x00 {
			return nil
		}
	}
	return fmt.Errorf("bug - invalid OID bytes - all 0x00")
}

/// Card ops ///////////////////////////////////////////////////////////////////

var (
	ErrCardExists   = fmt.Errorf("index.Card: card exists.")
	ErrCardNotFound = fmt.Errorf("index.Card: card for oid not found.")
)

func AddOrUpdateCard(path string, oid *OID, file string, tbm, sbm bitmap.Bitmap) (Card, bool, error) {

	card, newCard, e := getOrCreateCard(path, oid)
	if e != nil {
		return nil, false, e
	}

	if ok, e := card.AddPath(file); e != nil {
		return card, newCard, e
	} else if newCard && !ok {
		return card, newCard, fmt.Errorf("bug - index.AddOrUpdateCard: path not added on new card")
	}

	if e := card.SetTags(tbm); e != nil {
		return card, newCard, e
	}

	if e := card.SetSystemics(sbm); e != nil {
		return card, newCard, e
	}

	// REVU ok can serve as 'updated' for returning to gart/add.
	//      a new card certainly will return true
	//      a modified existing will also return true
	if ok, e := card.Save(); e != nil {
		return card, newCard, e
	} else if newCard && !ok {
		return nil, false, fmt.Errorf("bug - index.AddOrUpdateCard: path not added on new card")
	}

	return card, newCard, nil
}

func getOrCreateCard(path string, oid *OID) (Card, bool, error) {

	var cardfile = cardfilePath(path, oid)
	if !cardfileExists(cardfile) {
		dir := filepath.Dir(cardfile)
		if e := os.MkdirAll(dir, fs.DirPerm); e != nil {
			return nil, false, fmt.Errorf("bug - index.GetOrCreateCard: os.Mkdirall: %s", e)
		}
		return newCard0(oid, cardfile), true, nil
	}
	card, e := ReadCard(cardfile)
	return card, false, e
}

// REVU this expect a validated oid. Don't want to constantly verify.
// cpath has the card path, cfile is the full card path, including
func cardfilePath(path string, oid *OID) string {
	cpath := filepath.Join(path, "index/cards", fmt.Sprintf("%x", oid.dat[0]))
	return filepath.Join(cpath, fmt.Sprintf("%x.card", oid.dat[1:]))
}

func cardfileExists(cardfile string) bool {
	if _, e := os.Stat(cardfile); e != nil && os.IsNotExist(e) {
		return false
	} else if e != nil {
		panic(fmt.Errorf("bug - index.CardExists: %e", e))
	}
	return true
}
