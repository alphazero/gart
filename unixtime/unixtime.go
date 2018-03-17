// Doost!

// unixtime defines a wrapper for unsigned 32bit timestamps with second
// precision and various calendar helpers.
// REVU change the package name -- too long and not general for top level
package unixtime

import (
	"fmt"
	"strings"
	"time"
)

// Time just holds the unix seconds.
type Time uint32

const TimeSize = 4

// basically a semantic convenience method
func (t Time) Timestamp() uint32 { return uint32(t) }

// Returns the std-lib time.Time for this unixtime.Time.
func (t Time) StdTime() time.Time { return time.Unix(int64(t), 0) }

// Returns the mmm-dd-yyyy (ex: mar-21-2018)
func (t Time) Date() string {
	y, m, d := t.StdTime().Date()
	return fmt.Sprintf("%s-%02d-%4d", strings.ToLower(m.String()[:3]), d, y)
}

// REVU the local issue needs some thinking
// TODO try this late night before midnight
func timenow() time.Time {
	return time.Now()
}

// Returns unix Time for time.Now()
func Now() Time {
	return Time(uint32(timenow().Unix()) & 0xffffffff)
}
