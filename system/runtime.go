// Doost!

// system/runtime.go contains all runtime specific bits of the gart/system package.

package system

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/errors"
)

// REVU basic logging at some point is TODO

// change with go run|build -ldflags="-X github.com/alphazero/gart/system.DebugFlag=true" <main.go>
var Debug bool // = true
var DebugFlag string

// Set by init(), either /dev/null or stderr (if Debug is true)
//var DebugWriter io.Writer

// panics if any errors are encountered.
func init() {
	var err = errors.For("system.init")

	if DebugFlag == "true" {
		Debug = true
	}
	if Debug { // distinct if check as sometimes Debug is set to true w/out build flags
		debug.Writer = os.Stderr
	} else {
		var e error
		debug.Writer, e = os.Open(os.DevNull)
		if e != nil {
			panic(err.FaultWithCause(e, "unexpected error"))
		}
	}

	// gart repo is in user's home dir.
	user, e := user.Current()
	if e != nil {
		panic(err.FaultWithCause(e, "unexpected error"))
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
	debug.Printf("debug.Writer is %s", debug.Writer.(*os.File).Name())
	debug := debug.For("system.init")
	debug.Printf("--- system.init() ---------------------")
	debug.Printf("RepoPath:          %q", RepoPath)
	debug.Printf("TagsPath:          %q", TagsPath)
	debug.Printf("TagDictionaryPath: %q", TagDictionaryPath)
	debug.Printf("IndexPath:         %q", IndexPath)
	debug.Printf("IndexCardsPath:    %q", IndexCardsPath)
	debug.Printf("IndexTagmapsPath:  %q", IndexTagmapsPath)
	debug.Printf("ObjectIndexPath:   %q", ObjectIndexPath)
	debug.Printf("--- system.init() ------------- end ---")
	// end sanity check

	// errors

	ErrIndexExist = errors.Error("%q exists", IndexPath)
	ErrIndexNotExist = errors.Error("%q does not exist", IndexPath)

}
