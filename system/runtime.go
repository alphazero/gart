// Doost!

// system/runtime.go contains all runtime specific bits of the gart/system package.

package system

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

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

	// sanity & fat-finger checking. various gart components remove directories
	// and nested content. A prior bug had joined various paths (above) to user's
	// home. The only assumption here below is that gart repo dir is called .gart
	// and that it is directly nested in user-home (obtained from OS). So even if
	// we got that assumption wrong (i.e. gart's repo dir is called something else
	// or moved somewhere else) no component of gart will ever delete non-gart
	// fires or directories. (No, unit-testing is not sufficient. system package
	// however is included by every other package (except gart/syslib/errors) and
	// this one-time runtime check will prevent un-necessary grief.)
	var safePrefix = filepath.Join(user.HomeDir, ".gart")
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
	Debugf("system/runtime.go: system.init() ---------------------")
	Debugf("RepoPath:          %q", RepoPath)
	Debugf("TagsPath:          %q", TagsPath)
	Debugf("TagDictionaryPath: %q", TagDictionaryPath)
	Debugf("IndexPath:         %q", IndexPath)
	Debugf("IndexCardsPath:    %q", IndexCardsPath)
	Debugf("IndexTagmapsPath:  %q", IndexTagmapsPath)
	Debugf("ObjectIndexPath:   %q", ObjectIndexPath)
	Debugf("system/runtime.go: system.init() ------------- end ---")
	// end sanity check

	// errors

	ErrIndexExist = errors.Error("%q exists", IndexPath)
	ErrIndexNotExist = errors.Error("%q does not exist", IndexPath)

}

func Debugf(fmtstr string, a ...interface{}) {
	if !Debug {
		return
	}
	fmt.Fprintf(os.Stdout, "debug - "+fmtstr+"\n", a...)
}
