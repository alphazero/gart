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

/* XXX

.gart/
	index/
		tagdict.dat
		objects.idx
		cards/
		tagmaps/
*/

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
	IndexPath = filepath.Join(user.HomeDir, IndexDir)
	// both objects.idx and tagdict.dat are in .gart/index/
	ObjectIndexPath = filepath.Join(IndexPath, ObjectIndexFilename)
	TagDictionaryPath = filepath.Join(IndexPath, TagDictionaryFilename)
	// mutli-file card and tagmap files are nested in .gart/index/<idx-type>
	IndexCardsPath = filepath.Join(IndexPath, "cards")
	IndexTagmapsPath = filepath.Join(IndexPath, "tagmaps")

	// errors

	ErrIndexExist = errors.Error("%q exists", IndexPath)
	ErrIndexNotExist = errors.Error("%q does not exist", IndexPath)

	Debugf("system/runtime.go: system.init() ---------------------")
	Debugf("RepoPath:          %q", RepoPath)
	Debugf("IndexPath:         %q", IndexPath)
	Debugf("IndexCardsPath:    %q", IndexCardsPath)
	Debugf("IndexTagmapsPath:  %q", IndexTagmapsPath)
	Debugf("ObjectIndexPath:   %q", ObjectIndexPath)
	Debugf("TagDictionaryPath: %q", TagDictionaryPath)
	Debugf("system/runtime.go: system.init() ------------- end ---")
}

func Debugf(fmtstr string, a ...interface{}) {
	if !Debug {
		return
	}
	fmt.Fprintf(os.Stdout, "debug - "+fmtstr+"\n", a...)
}
