// Doost

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

/// system wide consts & vars /////////////////////////////////////////////////

// permissions of gart fs artifacts
const (
	dirPerm  = 0755
	filePerm = 0644
)

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
	if e := walkDirs(pi, verifyDir); e != nil {
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
	if e := walkDirs(pi, func(path string) error {
		return os.Mkdir(path, dirPerm)
	}); e != nil {
		panic(e)
	}

	// TODO create minimal/initial gart files
	// tagsdef
	// index
	// etc
	panic("initGart - is incomplete")
}

/// REVU gart/repo or gart/files is better place for these

// verify path, that it is a directory, and that the permissions are dirPerm
func verifyDir(path string) error {
	fi, e := verifyFileOrDir(path, dirPerm)
	if e != nil {
		return e
	}
	if !fi.IsDir() {
		return fmt.Errorf("verify error - not a directory: %s", path)
	}
	return nil
}

// verify path, that it is a regular file, and that the permissions are filePerm
func verifyFile(path string) error {
	fi, e := verifyFileOrDir(path, dirPerm)
	if e != nil {
		return e
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("verify error - not a regular file: %s", path)
	}
	return nil
}

// polymorphism anyone ..
func verifyFileOrDir(path string, expectedPerm os.FileMode) (os.FileInfo, error) {
	fi, e := os.Stat(path)
	if e != nil {
		return fi, fmt.Errorf("verify error - %s", e)
	}
	if perm := fi.Mode() & os.ModePerm; perm != expectedPerm {
		return fi, fmt.Errorf("verify error - invalid permission: %o %s", perm, path)
	}
	return fi, nil
}

// iterates over gartDirs and applies function fn.
// first error encountered is returned.
func walkDirs(pi processInfo, fn func(string) error) error {
	for _, dir := range gartDirs {
		path := filepath.Join(pi.user.HomeDir, dir)
		if e := fn(path); e != nil {
			return e
		}
	}
	return nil
}
