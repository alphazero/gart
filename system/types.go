// Doost!

package system

/// Runtime Context ////////////////////////////////////////////////////////////

// TODO
type Context struct {
}

/// Object Identity ////////////////////////////////////////////////////////////

// Size of binary representation of OID in bytes.
const OidSize = 32

// Object Identity is used to uniquely identity an archived content.
type Oid struct {
	dat [OidSize]byte
}

// NewOid expects a slice of atleast OidSize bytes, which will be validated
// and then copied for the allocated Oid.
//
// As this is a system function, all errors are treated as bugs.
//
// Returns (nil, BugInvalidArg) if the slice length is not as specified.
// Returns (nil, BugInvalidOidBytesData) if the minimal validation fails.
func NewOid(bytes []byte) (*Oid, error) {
	if len(bytes) < OidSize {
		return nil, BugInvalidArg
	}
	// sans actual content to verify, the minimal validation of the data
	// is that the slice can not possibly be all 0x00
	for _, b := range bytes {
		if b != 0x00 {
			goto valid
		}
	}
	// if reached, it was all 0x00
	return nil, BugInvalidOidBytesData

valid:
	var oid Oid
	copy(oid.dat[:], bytes[:OidSize])
	return &oid, nil
}

func (oid *Oid) String() string { return fmt.Sprintf("Oid:%x", oid.dat) }
