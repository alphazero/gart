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
	excluded() []string
	included() []string
}

type query struct {
	include map[string]struct{}
	exclude map[string]struct{}
}

func (q *query) Build() Query {
	return q
}

func (q query) included() []string {
	var tags = make([]string, len(q.include))
	var i int
	for k, _ := range q.include {
		tags[i] = k
		i++
	}
	return tags
}
func (q query) excluded() []string {
	var tags = make([]string, len(q.exclude))
	var i int
	for k, _ := range q.exclude {
		tags[i] = k
		i++
	}
	return tags
}

var _ QueryBuilder = &query{}

func NewQuery() *query {
	return &query{
		include: make(map[string]struct{}),
		exclude: make(map[string]struct{}),
	}
}

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
