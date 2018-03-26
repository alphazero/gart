// Doost!

package system

import (
	"github.com/alphazero/gart/syslib/errors"
)

/// defined errors /////////////////////////////////////////////////////////////

var (
	ErrTagNotFound = errors.Error("Tag for name not found")
)

/// defined bugs ///////////////////////////////////////////////////////////////

var (
	BugInvalidOidBytesData = errors.Bug("invalid oid bytes data")
)

/// defined faults /////////////////////////////////////////////////////////////

var (
	FaultOsRename = errors.Fault("os.Rename error")
)
