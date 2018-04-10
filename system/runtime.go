// Doost!

// system/runtime.go contains all runtime specific bits of the gart/system package.

package system

import (
	"os"
	"os/user"

	"github.com/alphazero/gart/repo"
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

	// repo.Paths
	repo.InitPaths(user.HomeDir)

	debug.Printf("debug.Writer is %s", debug.Writer.(*os.File).Name())
	debug := debug.For("system.init")
	debug.Printf("--- system.init() ---------------------")
	debug.Printf("RepoPath:          %q", repo.RepoPath)
	debug.Printf("TagsPath:          %q", repo.TagsPath)
	debug.Printf("TagDictionaryPath: %q", repo.TagDictionaryPath)
	debug.Printf("IndexPath:         %q", repo.IndexPath)
	debug.Printf("IndexCardsPath:    %q", repo.IndexCardsPath)
	debug.Printf("IndexTagmapsPath:  %q", repo.IndexTagmapsPath)
	debug.Printf("ObjectIndexPath:   %q", repo.ObjectIndexPath)
	debug.Printf("--- system.init() ------------- end ---")
	// end sanity check

	// errors

	ErrIndexExist = errors.Error("%q exists", repo.IndexPath)
	ErrIndexNotExist = errors.Error("%q does not exist", repo.IndexPath)

}
