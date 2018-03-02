// Doost!

package main

import (
	"fmt"
	"io"
	"os"
	"time"
	"unsafe"
)

const fname = "tempfile"

type Tag struct {
	flags  byte
	id     uint32
	refcnt uint32
	val    string // REVU TODO must insure that the byte len is < 251
}

func (r Tag) String() string {
	return fmt.Sprintf("%08b %d %d %q", r.flags, r.id, r.refcnt, r.val)
}

func (t *Tag) Decode(b []byte) (int, error) {
	if len(b) < 6 {
		return 0, fmt.Errorf("Tag.Decode - underflow - len: %d\n", len(b))
	}

	t.flags = b[0]
	t.id = *(*uint32)(unsafe.Pointer(&b[1]))
	t.refcnt = *(*uint32)(unsafe.Pointer(&b[5]))
	vlen := b[9]
	n := 10 + int(vlen)
	//	fmt.Printf("vlen %d n %d\n", vlen, n)
	t.val = string(b[10:n])
	return n, nil

}

func readTags(fname string) ([]Tag, error) {
	buf, e := readFile(fname)
	if e != nil {
		return nil, e
	}
	fmt.Printf("buf-len: %d\n", len(buf))

	var rcnt = *(*int)(unsafe.Pointer(&buf[0]))
	var off int = 8
	fmt.Printf("rcnt: %d\n", rcnt)

	// REVU assume 80% load factor to prevent a doubling resize
	var mapsize = int(float64(rcnt) * 1.25)
	var tags = make([]Tag, rcnt*2)
	var tagmap = make(map[string]int, mapsize)
	var i int
	for off < len(buf) {
		var tag Tag
		rlen, e := tag.Decode(buf[off:])
		if e != nil {
			return nil, e
		}
		// REVU we want to set tag.id = i here. TODO
		tagmap[tag.val] = off
		off += rlen
		//		tags[i] = tag
		i++
		//		tags = append(tags, tag)
		//		fmt.Printf("%v\n", tag)
	}
	return tags, nil
}
func readFile(fname string) ([]byte, error) {

	fi, e := os.Stat(fname)
	if e != nil {
		return nil, e
	}

	file, e := os.OpenFile(fname, os.O_RDONLY, 0644)
	if e != nil {
		return nil, e
	}
	defer file.Close()

	bufsize := fi.Size()
	buf := make([]byte, bufsize)

	n, e := io.ReadFull(file, buf)
	if e != nil {
		return buf, e
	}
	if n != int(bufsize) {
		panic("bug")
	}

	return buf, nil
}

/// main //////////////////////////////////////////////////////////////////////

func benchcomp(start, delta int64, n int) (time.Duration, float64, float64) {
	nspo := float64(delta) / float64(n)
	ops := float64(1000000000) / nspo
	return time.Duration(delta), nspo, ops
}

func main() {

	var start = time.Now().UnixNano()

	tags, e := readTags(fname)
	if e != nil {
		exitOnError(e)
	}
	var delta = time.Now().UnixNano() - start

	n := len(tags)
	dt, nspo, ops := benchcomp(start, delta, n)
	fmt.Printf("n:%d delta:%v ns/op:%f ops/s:%f\n", n, dt, nspo, ops)
}

/// helpers ///////////////////////////////////////////////////////////////////

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
