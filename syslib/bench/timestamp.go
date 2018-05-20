// Doost!

package bench

import (
	"fmt"
	"time"
)

type timestamp int64

func NewTimestamp() timestamp { return timestamp(time.Now().UnixNano()) }

func (t0 *timestamp) mark() int64 {
	var now = time.Now().UnixNano()
	var dt = now - int64(*t0)
	*t0 = timestamp(now)
	return dt
}

func (t0 *timestamp) Mark(s string) time.Duration {
	dt := time.Duration(t0.mark())
	fmt.Printf("time-mark: %s - dt:%s\n", s, dt)
	return dt
}

func (t0 *timestamp) MarkN(s string, ops int) (dt, dtpo time.Duration) {
	d := t0.mark()
	dt = time.Duration(d)
	dtpo = time.Duration(int64(float64(d) / float64(ops)))
	fmt.Printf("time-mark: %s - dt:%s t/op:%s\n", s, dt, dtpo)
	return
}
