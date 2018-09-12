// Doost!

package bench

import (
	"fmt"
	"time"
)

type timestamp time.Time

func NewTimestamp() timestamp { return timestamp(time.Now()) }

func (t0 *timestamp) mark() time.Duration {
	dt := time.Since(time.Time(*t0))
	*t0 = timestamp(time.Now())
	return dt
}

func (t0 *timestamp) Mark(s string) time.Duration {
	dt := t0.mark()
	fmt.Printf("time-mark: %s - dt:%s\n", s, dt)
	return dt
}

func (t0 *timestamp) MarkN(s string, ops int) (dt, dtpo time.Duration) {
	dt = t0.mark()
	dtpo = time.Duration(int64(float64(dt) / float64(ops)))
	fmt.Printf("time-mark: %s - dt:%s t/op:%s\n", s, dt, dtpo)
	return
}
