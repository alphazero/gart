// Doost!

package systemic

import (
	"fmt"
	"strings"
	"time"
)

/// systemics //////////////////////////////////////////////////////////////////

func ExtTag(name string) string  { return fmt.Sprintf("systemic:ext:%s", name) }
func TypeTag(name string) string { return fmt.Sprintf("systemic:type:%s", name) }
func TodayTag() string           { return DayTag(time.Now()) }

func DayTag(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("systemic:day:%s-%02d-%d", strings.ToLower(m.String()[:3]), d, y)
}
