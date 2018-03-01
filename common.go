// Doost

package main

import (
	"os"

	"github.com/alphazero/gart/fs"
)

/// system wide consts & vars /////////////////////////////////////////////////

// gart's top level (minimal) directory structure
const (
	gartDir  = ".gart"
	tagsDir  = ".gart/tags"
	pathsDir = ".gart/paths"
	indexDir = ".gart/index"
	cardsDir = ".gart/index/cards"
)

var gartDirs = []string{
	gartDir,
	tagsDir,
	pathsDir,
	indexDir,
	cardsDir,
}

// gart metadata and index files
const (
	tagsdefFile  = "tags/tagsdef"
	tagIndexFile = "index/tags.idx"
	devicesFile  = "path/devices"
)

/// initialization & process boot /////////////////////////////////////////////

// Verify existing gart repo or initialize the minimal structure.
//
// If top level gart repo exists but any of the mininal files or dirs
// is missing, treat it as a corrupted repo and panic.
//
// Check permissions and if not as expected, treat it as a corrupted repo and panic.
func initOrVerifyGart(pi processInfo) error {
	// is this the first use?
	if _, err := os.Stat(pi.gartDir); os.IsNotExist(err) {
		initGart(pi)
	}

	return verifyGartRepo(pi)

}

func verifyGartRepo(pi processInfo) error {
	// verify directory structure
	if e := fs.WalkDirs(pi.user.HomeDir, gartDirs, fs.VerifyDir); e != nil {
		return e
	}

	// verify .gart/ minimal files
	// .gart/tag/tags
	// .gart/path/devices
	panic("verifyGartRepo - is imcomplete")
}

// panics if init is called for an already initialized gart (REVU for now).
// Errors should be considered fail-stop.
func initGart(pi processInfo) error {
	if e := fs.WalkDirs(pi.user.HomeDir, gartDirs, func(path string) error {
		return os.Mkdir(path, fs.DirPerm)
	}); e != nil {
		panic(e)
	}

	// TODO create minimal/initial gart files
	// tagsdef
	// index
	// etc
	panic("initGart - is incomplete")
}
