// Doost

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/tag"
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
	tagsdefFile         = "tags/TAGS"
	objecttagsIndexFile = "index/OID-TAGS"
	devicesFile         = "paths/DEVICES"
)

/// gart initialization & process boot ////////////////////////////////////////

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

// panics if init is called for an already initialized gart (REVU for now).
// Errors should be considered fail-stop.
func initGart(pi processInfo) error {
	if e := fs.WalkDirs(pi.user.HomeDir, gartDirs, func(path string) error {
		return os.Mkdir(path, fs.DirPerm)
	}); e != nil {
		panic(e)
	}

	// TODO create minimal/initial gart files
	var fname string

	// tag definitions file created on tagmap load.
	// Note that tag.LoadMap immediately closes file.
	fname = filepath.Join(pi.gartDir, tagsdefFile)
	_, e := tag.LoadMap(fname, true)
	if e != nil {
		return e
	}

	// oid-tags index
	fname = filepath.Join(pi.gartDir, objecttagsIndexFile)
	fmt.Fprintf(os.Stderr, "WARNING - gart.initGart: %s creation is TODO\n", fname)

	// devices index
	fname = filepath.Join(pi.gartDir, devicesFile)
	fmt.Fprintf(os.Stderr, "WARNING - gart.initGart: %s creation is TODO\n", fname)

	return nil
}

func verifyGartRepo(pi processInfo) error {
	// verify directory structure
	if e := fs.WalkDirs(pi.user.HomeDir, gartDirs, fs.VerifyDir); e != nil {
		return e
	}

	files := []string{tagsdefFile, objecttagsIndexFile, devicesFile}
	for _, file := range files {
		fname := filepath.Join(pi.gartDir, file)
		// verify
		if e := fs.VerifyFile(fname); e != nil {
			return e
		}
	}
	return nil
}
