// Doost!

package index

import (
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/system"
	"github.com/alphazero/gart/system/systemic"
)

/// Query //////////////////////////////////////////////////////////////////////

type QueryBuilder interface {
	IncludeTags(tag ...string) *query
	ExcludeTags(tag ...string) *query
	OfType(otype system.Otype) *query
	ExcludeType(otype system.Otype) *query
	WithExtension(ext string) *query
	ExcludeExtension(ext string) *query
	Build() Query
}

type Query interface {
	asQuery() *query
}

type query struct {
	include map[string]struct{}
	exclude map[string]struct{}
}

func NewQuery() *query {
	return &query{
		include: make(map[string]struct{}),
		exclude: make(map[string]struct{}),
	}
}

func (q *query) Build() Query {
	var debug = debug.For("query.Build")
	debug.Printf("query")
	debug.Printf("-- include --")
	for k := range q.include {
		debug.Printf("\t%s", k)
	}
	debug.Printf("-- exclude --")
	for k := range q.exclude {
		debug.Printf("\t%s", k)
	}
	return q
}

func (q *query) asQuery() *query { return q }

func (q *query) IncludeTags(tags ...string) *query {
	for _, tag := range tags {
		q.include[tag] = struct{}{}
		delete(q.exclude, tag)
	}
	return q
}

func (q *query) ExcludeTags(tags ...string) *query {
	for _, tag := range tags {
		q.exclude[tag] = struct{}{}
		delete(q.include, tag)
	}
	return q
}

// REVU the following are not used -- find uses above directly.
//
// 2 concerns:
// 1 - find cmd is injecting the type/ext/date semantics. minor concern
//     is replicating the same elsewhere.
//
// 2 - more importantly, query is treating all tags uniformly and that
//     is losing bits of information possibly useful for query planning.
//
// TODO
//		1 - find should use the functions below.
//		2 - query struct should be semantic
//		3 - (long term) query planner.
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

func (q *query) WithExtension(ext string) *query {
	var tag = systemic.ExtTag(ext)
	q.include[tag] = struct{}{}
	delete(q.exclude, tag)
	return q
}

func (q *query) ExcludeExtension(ext string) *query {
	var tag = systemic.ExtTag(ext)
	q.exclude[tag] = struct{}{}
	delete(q.include, tag)
	return q
}
