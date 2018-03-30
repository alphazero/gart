// Doost!

package system

/// system wide constants //////////////////////////////////////////////////////

// Repository home, top-level directory, and well-defined index file  names
const (
	RepoDir               = ".gart"
	TagsDir               = "tags" // TODO deprecated
	IndexDir              = "index"
	ObjectIndexFilename   = "objects.idx"
	TagDictionaryFilename = "tagdict.dat"
)

// To support os portability these immutable system facts are vars.
// Initialized in (runtime.go) init().
var (
	RepoPath  string
	IndexPath string
	//	IndexObjectsPath string // TODO deprecated
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

const (
	OidSize        = 32  // bytes
	MaxTagNameSize = 255 // bytes not chars.
)
