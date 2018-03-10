// Doost!

package index

import (
	"fmt"
	"time"
)

/// Object IDs /////////////////////////////////////////////////////////////////

const (
	OidBytes = 32
)

type OID [OidBytes]byte

func NewOid(b []byte) (*OID, error) {
	if len(b) < OidBytes {
		return nil, fmt.Errorf("err - index.NewOid: buf len is %d", len(b))
	}
	var oid OID
	copy(oid[:], b[:OidBytes])
	if !oid.IsValid() {
		return nil, fmt.Errorf("err - index.NewOid: invalid OID: %x", oid)
	}
	return &oid, nil
}

// Anything other than an all-zero buffer is valid.
func (oid OID) IsValid() bool {
	for _, b := range oid {
		if b != 0x00 {
			return true
		}
	}
	return false
}

/// Object Index Card //////////////////////////////////////////////////////////

type Card interface {
	CreatedOn() time.Time // unix seconds precision
	UpdatedOn() time.Time // unix seconds precision
	Flags() uint32        // REVU: use semantic flags e.g. Card.HasDups() bool, etc.

	// Object ID is the Card entry's content hash
	Oid() OID
	//
	TagsBitmap() []byte // REVU this should be bitmap.Bitmap
	//
	SystemicBitmap() []byte // REVU also bitmap.Bitmap
	//
	Paths() []string
	//
	AddPath(fpath string) (bool, error)
	//
	RemovePath(fpath string) (bool, error)
	//
	UpdateUserTagBah(bitmap []byte)
	//
	Save(fname string) (bool, error)
	// XXX
	DebugStr() string
}
