// Friend

package digest

import (
	"crypto"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
)

// insure these cryptos are linked via import. Such a goofy language.
var (
	_ = sha1.New
	_ = sha256.New
	_ = sha512.New
)

// Default uses sha-1
func Compute(fname string) ([]byte, error) {
	return ComputeWith(fname, crypto.SHA1)
}

func ComputeWith(fname string, hash crypto.Hash) (md []byte, err error) {
	defer func(e *error) {
		r := recover()
		if r != nil {
			*e = fmt.Errorf("(recovered) %v", r)
		}
	}(&err)

	// note: hash.New() can panic if crypto package not linked.
	return compute(fname, hash.New())
}

func compute(fname string, h hash.Hash) ([]byte, error) {
	f, e := os.Open(fname)
	if e != nil {
		return nil, e
	}
	defer f.Close()

	if _, e := io.Copy(h, f); e != nil {
		return nil, e
	}
	return h.Sum(nil), nil
}
