package telemetrytest

import (
	"sync"
	"time"
)

// Clock is a deterministic clock for telemetry.StartTimer. Now returns the
// current time and then advances it by the step, so a timer started and stopped
// once measures exactly one step. The zero Clock starts at the zero time with a
// zero step. It is safe for concurrent use.
type Clock struct {
	mu   sync.Mutex
	now  time.Time
	step time.Duration
}

// NewClock returns a Clock at start that advances by step on every Now call.
func NewClock(start time.Time, step time.Duration) *Clock {
	return &Clock{now: start, step: step}
}

// Now returns the current time, then advances it by the step. Pass c.Now as the
// clock argument of telemetry.StartTimer.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.now
	c.now = c.now.Add(c.step)
	return t
}

// Advance moves the clock forward by d (backwards if d is negative).
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
