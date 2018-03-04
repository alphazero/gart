// Doost!

package tag

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var tempDir string

/// santa's little helpers ////////////////////////////////////////////////////

func init() {
	tempDir = os.TempDir()
	if _, e := os.Stat(tempDir); e != nil {
		panic(fmt.Sprintf("os.TempDir - err: %s", e))
	}
}

func remove(t *testing.T, fpath string) {
	if e := os.Remove(fpath); e != nil {
		t.Fatalf("unexpected - os.Remove - err: %s", e)
	}
}

func tagmapPath() string {
	const fname = "tagmap.dat"
	return filepath.Join(tempDir, fname)
}

/// test: creating and loading tagmaps ////////////////////////////////////////

func TestCreateTagmapFile(t *testing.T) {
	var fpath = tagmapPath()

	if e := createTagmapFile(fpath); e != nil {
		// NOTE subsequent tests depend on this function
		t.Fatalf("tag.CreateTagmapFile - err: %s", e)
	}
	remove(t, fpath)
}

// tagmap load is expected to open in exclusive mode and
// close the file on call return.
func TestLoadTagmapCreate(t *testing.T) {
	var fpath = tagmapPath()

	tmap, e := LoadTagmap(fpath, true)
	if e != nil {
		t.Fatalf("tag.TestLoadTagmapCreate - create: true - err: %s", e)
	}
	defer remove(t, fpath)

	// expect error since it already exists
	_, e = LoadTagmap(fpath, true)
	if e == nil {
		t.Fatalf("tag.TestLoadTagmapCreate - error expected")
	}

	// expect no error since create arg is false
	_, e = LoadTagmap(fpath, false)
	if e != nil {
		t.Fatalf("tag.TestLoadTagmapCreate - create: false - err: %s", e)
	}

	// check size
	{
		var expected = uint64(0)
		var have = tmap.Size()
		if have != expected {
			t.Fatalf("tagmap.Size() - expected:%d have:%d", expected, have)
		}
	}
}
