// Doost!

package system

/// system wide constants //////////////////////////////////////////////////////

/*
// Repository home, top-level directory, and well-defined index file  names
const (
	RepoDir               = ".gart"
	TagsDir               = "tags"
	IndexDir              = "index"
	ObjectIndexFilename   = "objects.idx"
	TagDictionaryFilename = "tagdict.dat"
)

// To support os portability these immutable system facts are vars.
// Initialized in (runtime.go) init().
var (
	RepoPath          string
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
*/
const (
	OidSize        = 32  // bytes
	MaxTagNameSize = 255 // bytes not chars. XXX deprecated
)
