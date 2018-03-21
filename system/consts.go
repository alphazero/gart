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
