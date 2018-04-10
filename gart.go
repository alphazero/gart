// Doost!

package gart

import (
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
)

func InitRepo(force bool) (bool, error) {
	var err = errors.For("gart.InitRepo")
	var debug = debug.For("gart.InitRepo")

	debug.Printf("in-args: force:%t", force)

	return false, err.NotImplemented()
}
