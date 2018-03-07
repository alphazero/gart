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
// REVU anything other than init should just call verifyGart TODO REMOVE THIS
func initOrVerifyGart(pi processInfo, silentInit bool) error {
	// is this the first use?
	if _, err := os.Stat(pi.gartDir); os.IsNotExist(err) && silentInit {
		initGart(pi, false, false)
	}

	return verifyGartRepo(pi)

}

// panics if init is called for an already initialized gart (REVU for now).
// Errors should be considered fail-stop.
func initGart(pi processInfo, force, silent bool) error {
try: // if forcing a re-init, we try twice
	for {
		e := fs.WalkDirs(pi.user.HomeDir, gartDirs, func(path string) error {
			return os.Mkdir(path, fs.DirPerm)
		})
		if e != nil {
			if force {
				if !silent {
					fmt.Fprintln(pi.meta, "gart-init: removing existing repo at %q", pi.gartDir)
				}
				if rme := os.RemoveAll(pi.gartDir); rme != nil {
					panic(rme) // bug or system error
				}
				force = false
				goto try
			}
			fmt.Sprintln(pi.meta, e)
			return fmt.Errorf("gart.InitGart: %s", e)
		}
		break
	}

	var fname string

	// tag definitions file created on tagmap load.
	// Note that tag.LoadMap immediately closes file.
	fname = filepath.Join(pi.gartDir, tagsdefFile)
	_, e := tag.LoadMap(fname, true)
	if e != nil {
		return e
	}

	// TODO create minimal/initial gart files
	// oid-tags index
	fname = filepath.Join(pi.gartDir, objecttagsIndexFile)
	fmt.Fprintf(os.Stderr, "WARNING - gart.initGart: %s creation is TODO\n", fname)

	// TODO create minimal/initial gart files
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
