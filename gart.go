// Doost!

package gart

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/repo"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

/// errors /////////////////////////////////////////////////////////////////////

var (
	ErrIgnoredPath = errors.Error("ignored path")
)

/// invariants /////////////////////////////////////////////////////////////////

// TODO .gartignore files and paths
var ignoredPaths = []string{".gart/", ".git/", ".git_vendor/"}
var ignoredExts = []string{".jar", ".pom, .bin, .class, .xml, .lock"}

/// stateless ops //////////////////////////////////////////////////////////////

func InitRepo(force bool) (bool, error) {
	//	var err = errors.For("gart.InitRepo")
	//	var debug = debug.For("gart.InitRepo")

	debug.Printf("in-args: force:%t", force)

	// initialize gart's repository
	if e := repo.Initialize(force); e != nil {
		return false, e
	}

	// initialize index
	// NOTE see comment in index.Initialize. 'force' is ignored.
	if e := index.Initialize(force); e != nil {
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
	// adds object to the index. strict flag only adds new objects; if false
	// new tags for existing objects are applied. in-arg spec semantics are per
	// the object type.
	//
	// Returns card for object, bool flag indicating if newly added, and nil
	// on success. On error, the card and flag values are undefined.
	AddObject(bool, system.Otype, string, ...string) (index.Card, bool, error)
	// Async processes the given query asynchronously, emitting selected objects
	// in the first returned channel, and any errors encountered in the second
	// channel.
	AsyncExec(query index.Query) (<-chan interface{}, <-chan error)

	Log() []string

	// Closes the session. If commit flag is true, changes made during the
	// session are committed. Otherwise, they are rolled back.
	Close(bool) error
}

type session struct {
	ctx           context.Context
	op            Op
	idx           index.IndexManager
	idxMode       index.OpMode
	interrupted   bool
	transactional bool
}

func OpenSession(ctx context.Context, op Op) (Session, error) {
	var err = errors.For("gart.OpenSession")
	var debug = debug.For("gart.OpenSession")
	debug.Printf("called - ctx:%v op:%08b", ctx, op)

	var idxMode index.OpMode
	var transactional bool
	switch op {
	case Add, Update, Compact, Tag:
		idxMode = index.Write
		transactional = true
	case Find:
		idxMode = index.Read
	}
	debug.Printf("idx opmode:%s", idxMode)

	idx, e := index.OpenIndexManager(idxMode)
	if e != nil {
		return nil, err.ErrorWithCause(e, "op:%s idxMode:%s", op, idxMode)
	}

	s := &session{
		ctx:           ctx,
		op:            op,
		idx:           idx,
		idxMode:       idxMode,
		transactional: transactional,
	}

	return s, nil
}

func (s *session) Close(commit bool) error {
	var err = errors.For("gart#session.Close")
	var debug = debug.For("gart#session.Close")
	debug.Printf("called - op:%s commit:%t s.transactional:%t", s.op, commit, s.transactional)

	if commit {
		// REVU for now ignore if commit is 'true' on non-transactional sessions
		if e := s.idx.Close(commit); e != nil {
			return err.ErrorWithCause(e, "on idx.Close(%t) - op:%s idxMode:%s", commit, s.op, s.idxMode)
		}
	} else {
		if s.transactional {
			if e := s.idx.Rollback(); e != nil {
				return err.ErrorWithCause(e, "on rollback - op:%s idxMode:%s", s.op, s.idxMode)
			}
		}
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
		if ignoreFile(path) {
			return nil, false, ErrIgnoredPath
		}
		return s.idx.IndexFile(strict, path, tags...)
	case system.URL, system.URI:
		return nil, false, err.InvalidArg("%s type not supported", otype)
	}
	panic(err.Bug("unreachable"))
}

var hiddenDir = []byte{os.PathSeparator, '.'}

// ignore returns true if file should be ignored.
func ignoreFile(path string) bool {
	// filter hidden paths
	if strings.Contains(path, string(hiddenDir)) {
		return true
	}

	// filter hidden files
	fname := filepath.Base(path)
	if fname[0] == '.' {
		return true
	}

	// filter if path contains ignored path element
	for _, s := range ignoredPaths {
		if strings.Contains(path, s) {
			return true
		}
	}

	// filter ignored extensions
	for _, s := range ignoredExts {
		if strings.HasSuffix(path, s) {
			return true
		}
	}

	return false
}

func (s *session) Log() []string {
	return []string{}
}

// TODO Select for query and modified signature.
func (s *session) AsyncExec(query index.Query) (<-chan interface{}, <-chan error) {
	var err = errors.For("gart#session.Exec")
	var debug = debug.For("gart#session.Exec")
	debug.Printf("called - query: %v", query)

	var oc = make(chan interface{}, 1)
	var ec = make(chan error, 1)

	go func() {
		oids, e := s.idx.Search(query)
		if e != nil {
			debug.Printf("err: %v", e)
			ec <- e
			return
		}
		debug.Printf("gart.Exec: found %v objects\n", len(oids))
		//		close(oc) // XXX
		//		close(ec) // XXX
		//	return

		for i, oid := range oids {
			select {
			case <-s.ctx.Done():
				debug.Printf("loading cards - interrupted")
				ec <- err.Error("interrupted (loaded %d of %d cards)", i, len(oids))
				s.interrupted = true
				return
			default:
				card, e := index.LoadCard(oid)
				if e != nil {
					ec <- err.ErrorWithCause(e, "on load of oid:%s", oid.Fingerprint())
					return
				}
				oc <- card
			}
		}
		close(oc)
		close(ec)
	}()
	return oc, ec
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
