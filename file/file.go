// Doost

// Package file has mostly helper functions dealing with files.
// Specific gart system files are handled in their respective packages
// e.g. gart/index/card.
package file

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

/// types /////////////////////////////////////////////////////////////////////

// Details encapsulates the necessary details for gart purposes.
// This structure is typically provided for user files.
type Details struct {
	Path   string // absolute path
	Dir    string // from root dir
	Name   string
	Ext    string
	Fstat  os.FileInfo
	Statfs syscall.Statfs_t
}

// REVU TODO filesystem details would be nice.
//			 also consider a Print func for full emit of details on multi-lines.
// String returns representation of Details suitable for logging.
func (s Details) String() string {
	str := fmt.Sprintf("file.Details name:%s path:%s dir:%s ext:%s", s.Name, s.Path, s.Dir, s.Ext)
	return str
}

func GetDetails(name string) (Details, error) {
	var details Details

	fpath, e := getAbsolutePath(name)
	if e != nil {
		return details, e
	}

	details.Path = fpath
	details.Name = filepath.Base(fpath)
	details.Ext = filepath.Ext(fpath)
	details.Dir = filepath.Dir(fpath)

	fstat, e := os.Stat(fpath)
	if e != nil {
		return details, e
	}
	details.Fstat = fstat

	if !fstat.Mode().IsRegular() {
		return details, fmt.Errorf("not a regular file - %s", name)
	}

	if e := syscall.Statfs(fpath, &details.Statfs); e != nil {
		return details, e
	}

	return details, nil
}

/// general helper functions //////////////////////////////////////////////////

func getAbsolutePath(name string) (string, error) {
	if filepath.IsAbs(name) {
		return name, nil
	}

	wd, e := os.Getwd()
	if e != nil {
		return "", e
	}

	fpath, e := filepath.Abs(filepath.Join(wd, name))
	if e != nil {
		return "", e
	}

	return fpath, nil
}
