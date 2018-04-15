// Doost!

package gart

import (
	"context"
	"path/filepath"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

var _ = errors.For
var _ = debug.For

/// stateless ops //////////////////////////////////////////////////////////////

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

// REVU rather not have cmds access index directly.
func FindCard(oidspec string) ([]index.Card, error) {
	return index.FindCard(oidspec)
}

func NewQuery() index.QueryBuilder { return index.NewQuery() }

/// Session ////////////////////////////////////////////////////////////////////

// Session represents a multi-op gart session.
type Session interface {
	// AddObject (strict, type, spec, tags...)
	AddObject(bool, system.Otype, string, ...string) (index.Card, bool, error)
	Exec(query index.Query) ([]index.Card, error) // TODO figure out the signature

	Log() []string
	Close() error
}

type session struct {
	ctx     context.Context
	op      Op
	idx     index.IndexManager
	idxMode index.OpMode
}

func OpenSession(ctx context.Context, op Op) (Session, error) {
	var err = errors.For("gart.OpenSession")
	var debug = debug.For("gart.OpenSession")
	debug.Printf("called - ctx:%v op:%08b", ctx, op)

	var idxMode index.OpMode
	switch op {
	case Add, Update, Compact, Tag:
		idxMode = index.Write
	case Find:
		idxMode = index.Read
	}

	idx, e := index.OpenIndexManager(index.Write)
	if e != nil {
		return nil, err.ErrorWithCause(e, "op:%s idxMode:%s", op, idxMode)
	}

	s := &session{
		ctx:     ctx,
		op:      op,
		idx:     idx,
		idxMode: idxMode,
	}

	// REVU important
	// TODO add session shutdown handler to context

	return s, nil
}

func (s *session) Close() error {
	var err = errors.For("gart#session.Close")
	var debug = debug.For("gart#session.Close")
	debug.Printf("called")

	if e := s.idx.Close(); e != nil {
		return err.ErrorWithCause(e, "op:%s idxMode:%s", s.op, s.idxMode)
	}
	return nil
}

func (s *session) AddObject(strict bool, otype system.Otype, spec string, tags ...string) (index.Card, bool, error) {
	var err = errors.For("gart#session.AddObject")
	var debug = debug.For("gart#session.AddObject")
	debug.Printf("called - strict:%t otype:%s spec:%q tags: %q", strict, otype, spec, tags)

	switch otype {
	case system.Text:
		return s.idx.IndexText(strict, spec, tags...)
	case system.File:
		path, e := filepath.Abs(spec)
		if e != nil {
			return nil, false, err.ErrorWithCause(e, "unexpected error on filepath.Abs")
		}
		return s.idx.IndexFile(strict, path, tags...)
	case system.URL, system.URI:
		return nil, false, err.InvalidArg("%s type not supported", otype)
	}
	panic(err.Bug("unreachable"))
}

func (s *session) Log() []string {
	return []string{}
}

func (s *session) Exec(query index.Query) ([]index.Card, error) {
	var err = errors.For("gart#session.Exec")
	var debug = debug.For("gart#session.Exec")
	debug.Printf("called - query: %v", query)

	oids, e := s.idx.Exec(query)
	if e != nil {
		return nil, e
	}
	debug.Printf("oids:%v", oids)
	return nil, err.NotImplemented()
}

/// Op /////////////////////////////////////////////////////////////////////////

type Op byte

const (
	_ Op = iota
	Add
	Remove
	Update
	Find // REVU diff between find and list is Find requires a session.
	List
	Tag
	Compact
)

func (v Op) String() string {
	switch v {
	case Add:
		return "Op:Add"
	case Remove:
		return "Op:Remove"
	case Update:
		return "Op:Update"
	case Find:
		return "Op:Find"
	case List:
		return "Op:List"
	case Tag:
		return "Op:Tag"
	case Compact:
		return "Op:Compact"
	}
	panic(errors.Bug("unknown gart.Op: %d", v))
}
