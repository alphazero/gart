// Doost

package main

import (
	"context"
	"fmt"
	//	"os"
	//	"sort"
	"strings"

	//	"github.com/alphazero/gart/bitmap"
	"github.com/alphazero/gart/fs"
	"github.com/alphazero/gart/gart"
	"github.com/alphazero/gart/tag"
)

/// flags and processing mode /////////////////////////////////////////////////

var option = struct {
	tags string
}{}

func init() {
	flags.StringVar(&option.tags, "tags", option.tags,
		"quoted comma separated list of tags")
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
// TODO replace with gart.AddOpContext
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

	// REVU a minor concern here: this bit of code is used in all cmds
	// with the exception of gart-init.
	if e := verifyGartRepo(pi); e != nil {
		fmt.Fprintln(state.pi.meta, e)
		return fmt.Errorf("fatal - gart repo not initialized. run 'gart-init'")
	}

	// tag map __________________________
	state.tagmap = loadTagMap(pi)

	// Don't Map.Add systemics here. Only user tags.
	// Skip if user didn't define any tags
	if option.tags == "" {
		goto volumes
	}
	state.tags = strings.Split(option.tags, ",")

	for i, tag := range state.tags {
		tag = strings.Trim(tag, " ")
		if _, _, e := state.tagmap.Add(tag); e != nil {
			return fmt.Errorf("err - gart-add.cmdPrepare: on add tag %q - %v", tag, e)
		}
		state.tags[i] = tag
	}

volumes:
	// TODO index:volumes ____________________
	// TODO index:card _______________________

	return nil
}

// command gart-add
// Returns output, error if any, and abort
func process(ctx context.Context, buf []byte) (output []byte, err error, abort bool) {

	state := getState(ctx)
	defer func() { state.items++ }()

	// TODO: address gart.Context - should be in State?
	var gartCtx gart.OpContext
	gartCtx.Tagmap = state.tagmap

	// REVU: better if it returns a gart.AddStat struct
	res := gart.AddObject(&gartCtx, string(buf), state.tags)
	opResult, ok := res.(*gart.AddOpResult)
	if !ok {
		panic("bug - gart.AddObject did not return *gart.AddOpResult - do output & abort")
	}
	if opResult.Fault() {
		panic("on process fault - do output & abort")
	}
	if opResult.Err() != nil {
		panic("on process err - do output = ... do not abort")
	}

	// index:card _______________________
	// check if card exists.
	// -> new: create card file.
	//
	// -> old: read card,
	//		dup or not, chec oid-tags, get bitmap and compare.
	//			-> diff: update oid-tags. REVU we need BAH.AND(old, new) here TODO
	//			-> same: NOP
	//		check paths.
	//		-> dup: update card with dup's path
	//		-> nop: user is adding the same file again, possibly just adding tags
	//				REVU this should be OK if a (new) flag --update-tags is provided.
	//					 use-case: we did a find . | gart-add and remembered we forgot some tags.

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

	output = fmtOutput("added %v", opResult.Card)

	return
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

/// santa's little helpers ////////////////////////////////////////////////////

func fmtOutput(fmtstr string, a ...interface{}) []byte {
	return []byte(fmt.Sprintf(fmtstr, a...))
}

// XXX this is temporary for dev-debug
func emitTags(state *State, md []byte, fds *fs.FileDetails, systemics []string) []byte {
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
