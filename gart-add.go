// Doost

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alphazero/gart/digest"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/tag"
)

/// flags and processing mode /////////////////////////////////////////////////

var option = struct {
	tags string
}{}

func init() {
	flags.StringVar(&option.tags, "tags", option.tags, "quoted comma separated list of tags")
}

// Check mandatory flags, etc.
func checkFlags() error {
	return nil
}

// Each process determines the run mode per its cmd-line options pattern
func processMode() Mode {
	if flags.NArg() == 0 {
		return Piped
	}
	return Standalone
}

/// command specific state ////////////////////////////////////////////////////

// struct encapsulates mutable and immutable process values.
type State struct {
	pi processInfo
	/* gart-add specific */
	tags   []string
	tagmap tag.Map
	items  int // files processed (regardless of completion stat)
}

/// command processing ////////////////////////////////////////////////////////

// pre:
// prepare for a run of gart-add process.
func cmdPrepare(state *State) error {

	var pi = state.pi

	// REVU cmds need something like mode since below is really true for all cmds
	if e := verifyGartRepo(pi); e != nil {
		fmt.Fprintln(state.pi.meta, e)
		return fmt.Errorf("fatal - gart repo not initialized. run 'gart-init'")
	}

	state.tagmap = loadTagMap(pi)

	// TODO open .gart/index/tags.idx in ? APPEND mode.
	// REVU lock it ?

	// Don't Map.Add systemics here. Only user tags.
	state.tags = strings.Split(option.tags, ",")
	for i, tag := range state.tags {
		fmt.Fprintf(state.pi.meta, "debug - gart-add.cmdPrepare: adding tag %q\n", tag)
		tag = strings.Trim(tag, " ")
		if _, e := state.tagmap.Add(tag); e != nil {
			return fmt.Errorf("err - gart-add.cmdPrepare: on add tag %q - %v", e)
		}
		state.tags[i] = tag
	}
	return nil
}

// command gart-add
// Returns output, error if any, and abort
func process(ctx context.Context, b []byte) (output []byte, err error, abort bool) {

	state := getState(ctx)
	defer func() { state.items++ }()

	fds, e := fs.GetFileDetails(string(b))
	if e != nil {
		return nil, e, false // we don't abort - next file may be ok
	}

	// REVU TODO need to check Compute. Only returning errors from OpenFile?
	//      If yes, then it must be a PathError and no need to check pe.Err
	//      just warn and return to continue
	md, e := digest.Compute(fds.Path)
	if e != nil {
		pe := e.(*os.PathError) // REVU counting on digest.Compute being straight up here ..
		if pe.Err.Error() == "permission denied" {
			output = []byte(fmt.Sprintf("warn - gart-add: skipping %q - %s",
				pe.Path, pe.Err)) // do not abort
			return
		}
		panic(fmt.Errorf("bug - digest.Compute returned error - %s", e))
	}

	// TODO
	//		- update .gart/paths (if required)
	//		  REVU check state to see if this has already been done
	//		- create tags bitmap REVU default tags (e.g. ext) + cmdline options
	//        if --tags "...." are spec'd in options, update tags/tagsdef
	//			REVU this includes updating frequency count of the tags
	// 		- create card file
	//		  REVU card may already exist.
	//		  REVU if --no-dups is specified, return nil error but emit msg to stderr
	// 		- append to index/TAGS
	//		  REVU state should have this file open in APPEND mode already.

	// XXX temporary

	// tags _____________________________

	systemics, e := addFileSystemicTags(state.tagmap, fds)
	if e != nil {
		panic(e)
	}
	if e := addFileTags(state.tagmap, state.tags...); e != nil {
		panic(e)
	}

	// TODO create bitmap for tags.

	// output
	output = emit(state, md, &fds, systemics)
	return
}

func emit(state *State, md []byte, fds *fs.FileDetails, systemics []string) []byte {
	sout := fmt.Sprintf("%x.. %s user:[ ", md[:4], fds.Name)
	for _, ut := range state.tags {
		sout += fmt.Sprintf("%q ", ut)
	}
	sout += fmt.Sprintf("] system:[ ")
	for _, st := range systemics {
		sout += fmt.Sprintf("%q ", st)
	}
	sout += fmt.Sprintf("]")
	return []byte(sout)
}

func addFileSystemicTags(tagmap tag.Map, fds fs.FileDetails) ([]string, error) {
	systemics := tag.AllSystemic(fds)
	e := addFileTags(tagmap, systemics...)
	return systemics, e
}

// Critical step here is incrementing the refcnts. The tag may already
// exist in the tagsdef so Add may be a NOP.
// TODO just have tag.Map.Incr or Add return the assigned id. it will be
//      required for creating the bitmap.
func addFileTags(tagmap tag.Map, tags ...string) error {

	for _, name := range tags {
		if _, e := tagmap.Add(name); e != nil {
			return fmt.Errorf("bug - gart-add: addFileTags: on Add %q - %v", name, e)
		}
		if _, e := tagmap.IncrRefcnt(name); e != nil {
			return fmt.Errorf("bug - gart-add: addFileTags: on IncrRefCnt %q - %v", name, e)
		}
	}
	return nil
}

// post:
func processDone(ctx context.Context) error {

	state := getState(ctx)

	// tagmap may return (false, nil), which indicates that this gart-add run
	// did not result in the defintion of a new tag. But it must never return
	// an error.
	_, e := state.tagmap.Sync()
	if e != nil {
		return fmt.Errorf("bug - gart-add: tag.Map.Sync returned error - %v", e)
	}

	// TODO close .gart/index/tags.idx in APPEND mode.
	// REVU unlock it ?

	return nil
}
