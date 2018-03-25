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

var Debug bool = true

// panics if any errors are encountered.
func init() {
	// gart repo is in user's home dir.
	user, e := user.Current()
	if e != nil {
		panic(errors.FaultWithCause(e, "runtime.init(): unexpected error"))
	}

	// initialize non-const system vars
	RepoPath = filepath.Join(user.HomeDir, RepoDir)
	IndexCardsPath = filepath.Join(RepoPath, IndexDir, "cards")
	IndexObjectsPath = filepath.Join(RepoPath, IndexDir, "objects")
	IndexTagmapsPath = filepath.Join(RepoPath, IndexDir, "tagmaps")

	Debugf("system.init() ---------------------")
	Debugf("RepoPath:         %q", RepoPath)
	Debugf("IndexCardsPath:   %q", IndexCardsPath)
	Debugf("IndexObjectsPath: %q", IndexObjectsPath)
	Debugf("IndexTagmapsPath: %q", IndexTagmapsPath)
}

func Debugf(fmtstr string, a ...interface{}) {
	if !Debug {
		return
	}
	fmt.Fprintf(os.Stdout, "debug - "+fmtstr+"\n", a...)
}
