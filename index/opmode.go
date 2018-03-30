// Doost!

package index

import (
	"github.com/alphazero/gart/syslib/errors"
)

/// op mode ////////////////////////////////////////////////////////////////////

// OpMode is a flag type indicating index file access modes. It is used by
// tagmaps and object-index files.
type OpMode byte

const (
	Read OpMode = 1 << iota
	Write
	Verify
	Compact
)

// panics on invalid opMode
func (m OpMode) verify() error {
	switch m {
	case Read:
	case Write:
	case Verify:
	case Compact:
	default:
		return errors.Bug("index.OpMode: unknown mode - %d", m)
	}
	return nil
}

// Returns string rep. of opMode
func (m OpMode) String() string {
	switch m {
	case Read:
		return "Read"
	case Write:
		return "Write"
	case Verify:
		return "Verify"
	case Compact:
		return "Compact"
	}
	panic(errors.Bug("index.OpMode: unknown mode - %d", m))
}
