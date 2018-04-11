// Doost!

package gart

import (
	. "context"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
)

var _ = errors.For
var _ = debug.For

// Session represents a multi-op gart session.
type Session interface {
	AddText(ctx Context, text string, tags ...string) (index.Card, error)
	AddFile(ctx Context, filename string, tags ...string) (index.Card, error)

	Log(ctx Context) []string
	Close(ctx Context) error
}

type session struct {
}

func OpenSession(ctx Context) Session {
	//	var err = errors.For("gart.OpenSession")
	var debug = debug.For("gart.OpenSession")
	debug.Printf("called - ctx:%v", ctx)

	s := &session{}

	return s
}

func (s *session) AddText(ctx Context, text string, tags ...string) (index.Card, error) {
	var err = errors.For("gart#session.AddText")
	var debug = debug.For("gart#session.AddText")
	debug.Printf("called - text:%q tags: %q")

	return nil, err.NotImplemented()
}
func (s *session) AddFile(ctx Context, filename string, tags ...string) (index.Card, error) {
	var err = errors.For("gart#session.Close")
	var debug = debug.For("gart#session.Close")
	debug.Printf("called - filename:%q tags: %q")

	return nil, err.NotImplemented()
}
func (s *session) Log(ctx Context) []string {
	return []string{}
}
func (s *session) Close(ctx Context) error {
	var err = errors.For("gart#session.Close")
	var debug = debug.For("gart#session.Close")
	debug.Printf("called")

	return err.NotImplemented()
}

func InitRepo(force bool) (bool, error) {
	//	var err = errors.For("gart.InitRepo")
	//	var debug = debug.For("gart.InitRepo")

	debug.Printf("in-args: force:%t", force)

	// TODO create/re-set repo root
	// REVU that should be in gart/repo

	// initialize index
	e := index.Initialize(force)
	if e != nil {
		return false, e
	}
	return true, nil
}
