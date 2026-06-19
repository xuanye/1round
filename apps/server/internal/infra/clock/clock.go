package clock

import "time"

type Clock interface {
	Now() time.Time
}

type UTCClock struct{}

func (UTCClock) Now() time.Time { return time.Now().UTC() }
