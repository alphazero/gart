// Doost!

package oidx

import (
	"fmt"

	"github.com/alphazero/gart/index"
)

var (
	ErrObjectNotFound = fmt.Errorf("object.idx: OID for key not found")
	ErrInvalidOid     = fmt.Errorf("object.idx: Invalid OID")
)

// NOTE this is just to nail the interface for object.idx index.
//      These functions will be called by top level functions in index package.
//      Specifically, OIdx.Add does not check if the OID is already registered.
//      Object (OID) uniqueness is determined by the Cards index.
type Objects interface {
	// Adds an OID to the object index.
	Register(*index.OID) (uint64, error)
	// Returns the OID for the given object key. Returns error if not found.
	GetOId(uint64) (*index.OID, error)
}
