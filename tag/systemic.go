// Doost!

// systemic.go: automatic tags defined for archived objects.

package tag

import (
	"fmt"
	"strings"
	"time"

	"github.com/alphazero/gart/fs"
)

const prefix = "gart:tag:"

func Systemic(fds fs.FileDetails) []string {
	return []string{
		ext(fds.Ext),
		today(),
	}
}

// All gart objects are tagged with the journal date. This function retuns
// a tag name of form "MMM-dd-YYYY" (e.g. MAR-21-2018).
func today() string {
	y, m, d := time.Now().Date()
	m3 := strings.ToUpper(m.String()[:3])
	return fmt.Sprintf("%s-%02d-%d", m3, d, y)
}

// REVU can a file be name "<something>." TODO check
func ext(ext string) string {
	s := prefix + "ext:"
	if len(ext) > 1 {
		s += ext[1:]
	}
	return s
}
