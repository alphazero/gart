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
	DirPerm  = 0755 // all dirs are  drwxr-xr-x
	FilePerm = 0644 // all files are -rw-r--r--
)

// crc table Qs
const (
	CRC64_Q = 0xdb032018da3a511e // NOTE made up 'db' blah - so it's NOT random
	CRC32_Q = 0xdb032018         // TODO research crc table selection
)

/// types //////////////////////////////////////////////////////////////////////

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

/// api ////////////////////////////////////////////////////////////////////////

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

// Creates a new file. In-arg ops is OR'd with std. create flags.
func OpenNewFile(fname string, ops int) (*os.File, error) {
	flags := os.O_CREATE | os.O_EXCL | os.O_SYNC
	return os.OpenFile(fname, flags|ops, FilePerm)
}

// Creates a new swap file. If the swap file already exists and
// 'abort' is true, it will return the *os.PathError (e.Err = os.ErrExist).
// Otherwise, the existing swap file will be deleted & recreated, setting
// the bool return param to true indicating that file already existed,
// e.g. (*fp, true, nil).
//
// Any other errors are returned with (nil, false, e) regardless of abort
// in-arg.
func OpenNewSwapfile(fname string, abort bool) (*os.File, bool, error) {
	var ops = os.O_WRONLY | os.O_APPEND
	var tries int
	var existed bool
try:
	if tries > 2 {
		return nil, existed, retryLimitError("OpenNewSwapfile")
	}

	file, e := OpenNewFile(fname, ops)
	if e != nil && os.IsExist(e) {
		existed = true
		if abort {
			return nil, existed, e
		}
		if rme := os.Remove(fname); rme != nil {
			err := fmt.Errorf("bug - fs.OpenNewSwapfile: on os.Remove - %s", rme)
			return nil, existed, err
		}
		tries++
		goto try // try again
	}
	return file, existed, nil
}

// panics on zerolen/empty fname
func SwapfileName(fname string) string {
	if fname == "" {
		panic("bug - SwapfileName - fname is zerolen")
	}

	// [.../]fname -> [.../].fname.swp
	var swapbase = fmt.Sprintf(".%s.swp", filepath.Base(fname))
	return filepath.Join(filepath.Dir(fname), swapbase)
}

func OpenReadOnly(fname string) (*os.File, error) {
	var flags = os.O_RDONLY | os.O_SYNC
	return os.OpenFile(fname, flags, 0)
}

// Exclusively fully reads the named file. File is closed on return.
func ReadFull(fname string) ([]byte, error) {

	fi, e := os.Stat(fname)
	if e != nil {
		return nil, e
	}

	var flags = os.O_RDONLY | os.O_SYNC
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

/// santa's little helpers /////////////////////////////////////////////////////

// verify path, that it is a directory, and that the permissions are perm
func VerifyDir(path string) error {
	fi, e := verifyFileOrDir(path, DirPerm)
	if e != nil {
		return e
	}
	if !fi.IsDir() {
		return fmt.Errorf("err - fs.verifyDir - not a directory: %s", path)
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
		return fmt.Errorf("err - fs.verifyFile - not a regular file: %s", path)
	}
	return nil
}

// Checks that the file or dir exists, and, that the fs objects permissions are
// as expected.
func verifyFileOrDir(path string, expectedPerm os.FileMode) (os.FileInfo, error) {
	fi, e := os.Stat(path)
	if e != nil {
		return fi, fmt.Errorf("err - fs.VerifyFileOrDir - %s", e)
	}
	if perm := fi.Mode() & os.ModePerm; perm != expectedPerm {
		return fi, fmt.Errorf("err - fs.verifyFileOrDir - invalid permission: %o %s", perm, path)
	}
	return fi, nil
}

// Returns a more informative error. in-arg 'fun' is the function name.
func retryLimitError(fun string) error {
	return fmt.Errorf("bug - fs.%s: retry limit exceeded", fun)
}
