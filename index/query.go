// Doost!

package index

import (
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

func (q *query) Build() Query { return q }

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
