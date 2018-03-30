// Doost!

package system

import (
	"github.com/alphazero/gart/syslib/errors"
)

/// defined errors /////////////////////////////////////////////////////////////

// These errors will be initialized by package init()
var (
	ErrIndexExist    error
	ErrIndexNotExist error
)

//var	ErrTagNotFound = errors.Error("Tag for name not found") // REVU ???

/// defined bugs ///////////////////////////////////////////////////////////////

var (
	BugInvalidOidBytesData = errors.Bug("invalid oid bytes data")
)

/// defined faults /////////////////////////////////////////////////////////////

var (
	FaultOsRename = errors.Fault("os.Rename error")
)
