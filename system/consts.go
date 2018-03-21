// Doost!

package system

/// system wide constants //////////////////////////////////////////////////////

// Repository home and top-level directory names
const (
	RepoDir  = ".gart"
	TagsDir  = "tags"
	IndexDir = "index"
)

// To support os portability these immutable system facts are vars.
// Initialized in (runtime.go) init().
var (
	RepoPath         string
	IndexCardsPath   string
	IndexObjectsPath string
	IndexTagmapsPath string
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
