package profile

import (
	"time"
)

const (
	HeaderName = "X-Gondola-Profile"
	Salt       = "gnd.la/app/profile.salt"
)

type Event struct {
	Started time.Time
	Ended   time.Time
	Notes   []string
}

func (e *Event) Elapsed() time.Duration {
	return e.Ended.Sub(e.Started)
}

type Timing struct {
	Name   string
	Events []*Event
}

func (t *Timing) Count() int {
	return len(t.Events)
}

func (t *Timing) Total() time.Duration {
	total := time.Duration(0)
	for _, v := range t.Events {
		total += v.Elapsed()
	}
	return total
}
