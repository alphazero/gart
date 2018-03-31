// Doost!

// system/runtime.go contains all runtime specific bits of the gart/system package.

package system

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/alphazero/gart/syslib/errors"
)

// REVU basic logging at some point is TODO
// REVU this can be set with go run/build, btw
var Debug bool = true

// panics if any errors are encountered.
func init() {
	// gart repo is in user's home dir.
	user, e := user.Current()
	if e != nil {
		panic(errors.FaultWithCause(e, "runtime.init(): unexpected error"))
	}

	/// initialize non-const system vars ////////////////////////////

	// paths
	RepoPath = filepath.Join(user.HomeDir, RepoDir)

	TagsPath = filepath.Join(RepoPath, TagsDir)
	TagDictionaryPath = filepath.Join(TagsPath, TagDictionaryFilename)

	IndexPath = filepath.Join(RepoPath, IndexDir)
	ObjectIndexPath = filepath.Join(IndexPath, ObjectIndexFilename)
	IndexCardsPath = filepath.Join(IndexPath, "cards")
	IndexTagmapsPath = filepath.Join(IndexPath, "tagmaps")

	// errors

	ErrIndexExist = errors.Error("%q exists", IndexPath)
	ErrIndexNotExist = errors.Error("%q does not exist", IndexPath)

	Debugf("system/runtime.go: system.init() ---------------------")
	Debugf("RepoPath:          %q", RepoPath)
	Debugf("TagsPath:          %q", TagsPath)
	Debugf("TagDictionaryPath: %q", TagDictionaryPath)
	Debugf("IndexPath:         %q", IndexPath)
	Debugf("IndexCardsPath:    %q", IndexCardsPath)
	Debugf("IndexTagmapsPath:  %q", IndexTagmapsPath)
	Debugf("ObjectIndexPath:   %q", ObjectIndexPath)
	Debugf("system/runtime.go: system.init() ------------- end ---")
}

func Debugf(fmtstr string, a ...interface{}) {
	if !Debug {
		return
	}
	fmt.Fprintf(os.Stdout, "debug - "+fmtstr+"\n", a...)
}
