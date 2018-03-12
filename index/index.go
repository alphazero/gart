// Doost!

package index

import (
	"fmt"
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
