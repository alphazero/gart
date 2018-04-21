// Doost!

package bench

import (
	"fmt"
	"time"
)

type timestamp int64

func NewTimestamp() timestamp { return timestamp(time.Now().UnixNano()) }

func (t0 *timestamp) Mark(s string) {
	now := time.Now().UnixNano()
	fmt.Printf(">>> step:%s - dt:%s\n", s, time.Duration(now-int64(*t0)))
	*t0 = timestamp(now)
}
