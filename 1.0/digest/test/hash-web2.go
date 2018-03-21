// Doost!

package main

import (
	"fmt"
	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
	"os"
)

const web2filename = "/Users/alphazero/Code/go/src/gart/res/webster2-words.txt"

// NOTE using webster 2.0 word list as 'tags', to determine if uint64 hash
//      has any collisions. This is to confirm the expectation that it should not
//      happen. Web2 has 235886 words. That seems to be well beyond range of expected
//      tags.
//
// TODO we use hash of tag names to construct a bitmap file for that tag in the
//      same manner as the card files.
func main() {
	fmt.Printf("Salaam!\n")

	var m = make(map[uint64]struct{})

	// load webster 2.0
	buf, e := fs.ReadFull(web2filename)
	if e != nil {
		exitOnError(e)
	}
	var line [64]byte
	var i int
	for _, b := range buf {
		if b == '\n' {
			if e := update(m, line[:i]); e != nil {
				exitOnError(e)
			}
			i = 0
			continue
		}
		line[i] = b
		i++
	}
}

func update(m map[uint64]struct{}, word []byte) error {
	hash := digest.Sum(word)
	key := digest.SumUint64(word)
	dir := key >> 56
	mkey := key & 0x00ffffffffffffff
	fmt.Printf("%016x gart/tag/%02x/%014x %s %x\n", key, dir, mkey, string(word), hash)
	if _, ok := m[mkey]; ok {
		return fmt.Errorf("collision: mkey:%016x\n", mkey)
	}
	m[mkey] = struct{}{}
	return nil
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "err - %v\n", e)
	os.Exit(1)
}
