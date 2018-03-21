// Doost!

package gart

import (
	"fmt"
	"io"

	"github.com/alphazero/gart/tag"
)

/// OpContext ///////////////////////////////////////////////////////////////////

// Context provides the bindings to various runtime components of gart.
type OpContext struct {
	Meta   io.Writer // out-of-band output stream
	Home   string    // gart repo root dir
	Tagmap tag.Map
}

/*
func NewOpContext(gartDir string, meta io.Writer) (*OpContext, error) {
	if meta == nil {
		return nil, fmt.Errorf("gart.NewOpContext: illegal arg - meta is nil")
	}

	ctx := &OpContext{
		meta:    meta,
		gartDir: gartDir,
	}
	return ctx, nil
}
*/
/// OpResult ////////////////////////////////////////////////////////////////////

// Base gart operation result provides means to determine operation's error,
// if any, and whether the error is non-recoverable fault.
type OpResult interface {
	Err() error  // Error, if any
	Fault() bool // indicates a non-recoverable error or bug - Err must non nil
}

// opResult satisfies the OpResult interface.
type opResult struct {
	err   error // Error, if any
	fault bool  // indicates a non-recoverable error or bug - Err must non nil
}

func (r *opResult) Err() error  { return r.err }
func (r *opResult) Fault() bool { return r.fault }

func (r *opResult) onError(e error, fmtstr string, a ...interface{}) *opResult {
	fmtstr0 := "err - gart.AddObject: " + fmtstr + " - %s"
	r.err = fmt.Errorf(fmtstr0, a...)
	return r
}

func (r *opResult) onBug(e error, fmtstr string, a ...interface{}) *opResult {
	fmtstr0 := "bug - gart.AddObject: " + fmtstr + " - %s"
	r.err = fmt.Errorf(fmtstr0, a...)
	r.fault = true
	return r
}
