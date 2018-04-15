// Doost!

package gart

import (
	"context"
	"path/filepath"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/systemic"
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

type query struct {
	include map[string]struct{}
	exclude map[string]struct{}
}

func Query() *query {
	return &query{
		include: make(map[string]struct{}),
		exclude: make(map[string]struct{}),
	}
}

func (q *query) WithTag(tags ...string) *query {
	for _, tag := range tags {
		q.include[tag] = struct{}{}
		delete(q.exclude, tag)
	}
	return q
}

func (q *query) ExcludeTag(tags ...string) *query {
	for _, tag := range tags {
		q.exclude[tag] = struct{}{}
		delete(q.include, tag)
	}
	return q
}

func (q *query) OfType(otype system.Otype) *query {
	var tag = systemic.TypeTag(otype.String())
	q.include[tag] = struct{}{}
	delete(q.exclude, tag)
	return q
}

func (q *query) ExcludeType(otype system.Otype) *query {
	var tag = systemic.TypeTag(otype.String())
	q.exclude[tag] = struct{}{}
	delete(q.include, tag)
	return q
}

func (q *query) WithExt(ext string) *query {
	var tag = systemic.ExtTag(ext)
	q.include[tag] = struct{}{}
	delete(q.exclude, tag)
	return q
}

func (q *query) ExcludeExt(ext string) *query {
	var tag = systemic.ExtTag(ext)
	q.exclude[tag] = struct{}{}
	delete(q.include, tag)
	return q
}

/// Session ////////////////////////////////////////////////////////////////////

// Session represents a multi-op gart session.
type Session interface {
	// AddObject (strict, type, spec, tags...)
	AddObject(bool, system.Otype, string, ...string) (index.Card, bool, error)
	Select(query *query) ([]index.Card, error) // TODO figure out the signature

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

func (s *session) Select(query *query) ([]index.Card, error) {
	var err = errors.For("gart#session.Select")
	var debug = debug.For("gart#session.Select")
	debug.Printf("called - query: %v", query)

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
