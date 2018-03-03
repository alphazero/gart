// Doost

// Package file has mostly helper functions dealing with files.
// Specific gart system files are handled in their respective packages
// e.g. gart/index/card.
package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

/// system wide ///////////////////////////////////////////////////////////////

// permissions of gart fs artifacts
const (
	DirPerm  = 0755
	FilePerm = 0644
)

// crc table Qs
const (
	CRC64_Q = 0xdb032018da3a511e
	CRC32_Q = 0xDB032018
)

/// types /////////////////////////////////////////////////////////////////////

// FileDetails encapsulates the necessary details for gart purposes.
// This structure is typically provided for user files.
type FileDetails struct {
	Path   string // absolute path
	Dir    string // from root dir
	Name   string
	Ext    string
	Fstat  os.FileInfo
	Statfs syscall.Statfs_t
}

// REVU TODO filesystem details would be nice.
//			 also consider a Print func for full emit of details on multi-lines.
// String returns representation of FileDetails suitable for logging.
func (s FileDetails) String() string {
	str := fmt.Sprintf("FileDetails name:%s path:%s dir:%s ext:%s", s.Name, s.Path, s.Dir, s.Ext)
	return str
}

func GetFileDetails(name string) (FileDetails, error) {
	var details FileDetails

	fpath, e := filepath.Abs(name)
	if e != nil {
		return details, e
	}

	fstat, e := os.Stat(fpath)
	if e != nil {
		return details, e
	}
	details.Fstat = fstat
	details.Path = fpath
	details.Name = filepath.Base(fpath)
	details.Ext = filepath.Ext(fpath)
	details.Dir = filepath.Dir(fpath)

	if !fstat.Mode().IsRegular() {
		return details, fmt.Errorf("not a regular file - %s", name)
	}

	if e := syscall.Statfs(fpath, &details.Statfs); e != nil {
		return details, e
	}

	return details, nil
}

/// santa's little helpers ////////////////////////////////////////////////////

// Exclusively fully reads the named file. File is closed on return.
func ReadFull(fname string) ([]byte, error) {

	fi, e := os.Stat(fname)
	if e != nil {
		return nil, e
	}

	var flags = os.O_EXCL | os.O_RDONLY | os.O_SYNC
	file, e := os.OpenFile(fname, flags, FilePerm)
	if e != nil {
		return nil, e
	}
	defer file.Close()

	bufsize := fi.Size()
	buf := make([]byte, bufsize)

	_, e = io.ReadFull(file, buf)
	if e != nil {
		return nil, e
	}

	return buf, e
}

// verify path, that it is a directory, and that the permissions are perm
func VerifyDir(path string) error {
	fi, e := verifyFileOrDir(path, DirPerm)
	if e != nil {
		return e
	}
	if !fi.IsDir() {
		return fmt.Errorf("verify error - not a directory: %s", path)
	}
	return nil
}

// verify path, that it is a regular file, and that the permissions are filePerm
func VerifyFile(path string) error {
	fi, e := verifyFileOrDir(path, FilePerm)
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
//func walkDirs(pi processInfo, fn func(string) error) error {
func WalkDirs(rootpath string, dirs []string, fn func(string) error) error {
	for _, dir := range dirs {
		path := filepath.Join(rootpath, dir)
		if e := fn(path); e != nil {
			return e
		}
	}
	return nil
}
