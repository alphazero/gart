// Doost!

package index

import (
	"fmt"
	"io"

	"github.com/alphazero/gart/syslib/errors"
)

type Paths struct {
	size   int
	buflen int
	head   *_LL
}

type _LL struct {
	path string
	next *_LL
}

func NewPaths() *Paths {
	return &Paths{
		size: 0,
		head: nil,
	}
}

func (v Paths) Size() int { return v.size }

func (v Paths) Buflen() int { return v.buflen }

func (v Paths) List() []string {
	var list = make([]string, v.size)
	var node = v.head
	var n int
	for node != nil {
		list[n] = node.path
		node = node.next
		n++
	}
	return list
}

func (p Paths) Print(w io.Writer) {
	fmt.Fprintf(w, "path-cnt:  %d\n", p.size)
	fmt.Fprintf(w, "buflen:    %d\n", p.buflen)
	if p.size > 0 {
		pathlist := p.List()
		for n, path := range pathlist {
			fmt.Fprintf(w, "\tpath[%d] [len:%3d] %s\n", n, len(path), path)
		}
	}
}

func (p *Paths) Remove(path string) (bool, error) {
	if path == "" {
		return false, errors.InvalidArg("Paths.Remove", "path", "zero-len")
	}
	node := p.head
	var prev *_LL
	for node != nil {
		if node.path == path {
			if prev == nil { // remove head
				p.head = node.next
			} else {
				prev.next = node.next
			}
			p.size--
			p.buflen -= len(path) + 1
			return true, nil
		}
		prev = node
		node = node.next
	}
	return false, nil
}

func (p *Paths) Add(path string) (bool, error) {
	if path == "" {
		return false, errors.InvalidArg("Paths.Add", "path", "zero-len")
	}
	if p.head == nil {
		p.head = &_LL{path, nil}
	} else {
		var node = p.head
		for {
			if node.path == path {
				return false, nil
			}
			if node.next == nil {
				node.next = &_LL{path, nil}
				break
			}
			node = node.next
		}
	}
	p.size++
	p.buflen += len(path) + 1
	return true, nil
}

// REVU copies the bytes so it is safe with mmap.
// REVU exported for testing TODO doesn't need to be exported
func (p *Paths) Decode(buf []byte) error {
	if buf == nil {
		return errors.InvalidArg("Paths.decode", "buf", "nil")
	}
	readLine := func(buf []byte) (int, []byte) {
		var xof int
		for xof < len(buf) {
			if buf[xof] == '\n' {
				break
			}
			xof++
		}
		return xof + 1, buf[:xof]
	}
	var xof int
	for xof < len(buf) {
		n, path := readLine(buf[xof:])
		p.Add(string(path))
		xof += n
	}
	return nil
}

func (v Paths) Encode(buf []byte) error {
	if buf == nil {
		return errors.InvalidArg("Paths.encode", "buf", "nil")
	}
	if len(buf) < v.Buflen() {
		return errors.InvalidArg("Paths.encode", "buf", "< path.buflen")
	}
	var xof int
	for _, s := range v.List() {
		copy(buf[xof:], []byte(s))
		xof += len(s)
		buf[xof] = '\n'
		xof++
	}
	return nil
}
