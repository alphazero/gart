// Doost!

package systemic

import (
	"fmt"
	"strings"
	"time"
)

/// systemics //////////////////////////////////////////////////////////////////
func GartTag() string            { return fmt.Sprintf("systemic:gart-object") }
func ExtTag(name string) string  { return fmt.Sprintf("systemic:ext:%s", strings.ToLower(name)) }
func TypeTag(name string) string { return fmt.Sprintf("systemic:type:%s", name) }
func TodayTag() string           { return DayTag(time.Now()) }

func DayTag(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("systemic:day:%s-%02d-%d", strings.ToLower(m.String()[:3]), d, y)
}
