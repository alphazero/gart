// Doost!

package repo

import (
	"path/filepath"
	"strings"

	"github.com/alphazero/gart/syslib/errors"
)

/// system wide constants //////////////////////////////////////////////////////

// Repository home, top-level directory, and well-defined index file  names
const (
	RepoDir               = ".gart" // REVU rename to repo.Dir
	TagsDir               = "tags"
	IndexDir              = "index"
	ObjectIndexFilename   = "objects.idx"
	TagDictionaryFilename = "tagdict.dat"
)

// To support os portability these immutable system facts are vars.
// Initialized in (runtime.go) init().
var (
	RepoPath          string // REVU rename to Path
	TagsPath          string
	IndexPath         string
	ObjectIndexPath   string
	TagDictionaryPath string
	IndexCardsPath    string
	IndexTagmapsPath  string
)

// permissions of gart file-system artifacts
const (
	DirPerm  = 0755 // all dirs are  drwxr-xr-x
	FilePerm = 0644 // all files are -rw-r--r--
)

// panics on error
func InitPaths(rootDir string) {
	RepoPath = filepath.Join(rootDir, RepoDir)

	TagsPath = filepath.Join(RepoPath, TagsDir)
	TagDictionaryPath = filepath.Join(TagsPath, TagDictionaryFilename)

	IndexPath = filepath.Join(RepoPath, IndexDir)
	ObjectIndexPath = filepath.Join(IndexPath, ObjectIndexFilename)
	IndexCardsPath = filepath.Join(IndexPath, "cards")
	IndexTagmapsPath = filepath.Join(IndexPath, "tagmaps")

	// sanity & fat-finger checking. various gart components remove directories
	// and nested content. A prior bug had joined various paths (above) to user's
	// home. The only assumption here below is that gart repo dir is called .gart
	// and that it is directly nested in user-home (obtained from OS). So even if
	// we got that assumption wrong (i.e. gart's repo dir is called something else
	// or moved somewhere else) no component of gart will ever delete non-gart
	// fires or directories. (No, unit-testing is not sufficient. system package
	// however is included by every other package (except gart/syslib/errors) and
	// this one-time runtime check will prevent un-necessary grief.)
	var safePrefix = filepath.Join(rootDir, ".gart")
	var paths = []string{
		TagsPath,
		TagDictionaryPath,
		IndexPath,
		ObjectIndexPath,
		IndexCardsPath,
		IndexTagmapsPath,
	}
	for i, path := range paths {
		if !strings.HasPrefix(path, safePrefix) {
			panic(errors.Fault("paths[%d]: %q is not safe!", i, path))
		}
	}
}
