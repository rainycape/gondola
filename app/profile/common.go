package profile

import (
	"time"
)

const (
	HeaderName = "X-Gondola-Profile"
	Salt       = "gnd.la/app/profile.salt"
)

type Event struct {
	Started time.Time `json:"s"`
	Ended   time.Time `json:"e"`
	Notes   []string  `json:"n"`
}

func (e *Event) Elapsed() time.Duration {
	return e.Ended.Sub(e.Started)
}

type Timing struct {
	Name   string   `json:"n"`
	Events []*Event `json:"e"`
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
