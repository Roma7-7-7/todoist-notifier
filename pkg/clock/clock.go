package clock

import (
	"time"
)

type Interface interface {
	Now() time.Time
}

type Clock struct {
	now func() time.Time
}

func NewClock() *Clock {
	return &Clock{
		now: time.Now,
	}
}

func NewZonedClock(location *time.Location) *Clock {
	return &Clock{
		now: func() time.Time {
			return time.Now().In(location)
		},
	}
}

func (c *Clock) Now() time.Time {
	return c.now()
}
