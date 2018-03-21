// Doost!

// systemic.go: automatic tags defined for archived objects.

package tag

import (
	"fmt"
	"strings"
	"time"

	"github.com/alphazero/gart/fs"
)

// REVU keep for now ..
const prefix = "gart:tag:"

func AllSystemic(fds *fs.FileDetails) []string {
	return []string{
		Ext(fds.Ext),
		Today(),
	}
}

// All gart objects are tagged with the journal date. This function retuns
// a tag name of form "MMM-dd-YYYY" (e.g. MAR-21-2018).
func Today() string {
	y, m, d := time.Now().Date()
	m3 := "day:" + strings.ToLower(m.String()[:3])
	return fmt.Sprintf("%s-%02d-%d", m3, d, y)
}

// Note: it is possible that a user may choose to define a tag that collides
// with an, e.g. '.txt.', extension. For now the 'ext:' prefix addresses such
// a case, but the query api ~is:
//
// 		gart-find --ext pdf --tags "...." # find all pdf objects + tags
//  or
// 		gart-find --no-ext --tags "...."  # find all objects with no extension + ..
//
// so even if user (for whatever reason) has applied e.g. '.txt' tag, it can
// not collide in the tag.Map. Of course, the prefix is necessary.
//
// ex: "ext:pdf" # .pdf extension
// ex: "ext:"    # no extension
func Ext(ext string) string {
	s := "ext:"
	if len(ext) > 1 {
		s += ext[1:]
	}
	return s
}
