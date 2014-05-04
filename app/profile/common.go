package profile

import (
	"time"
)

const (
	// HeaderName is the header used for transmitting profiling
	// information while profiling a remote application.
	HeaderName = "X-Gondola-Profile"
	// Salt is the salt used for signing the app secret for requesting
	// profiling information.
	Salt = "gnd.la/app/profile.salt"
)

// Event represents a finished event when its timing information
// and any notes it might have attached.
type Event struct {
	Started time.Time `json:"s"`
	Ended   time.Time `json:"e"`
	Notes   []string  `json:"n"`
}

// Ellapsed returns the time the event took.
func (e *Event) Elapsed() time.Duration {
	return e.Ended.Sub(e.Started)
}

// Timing represents a set of events of the same
// kind.
type Timing struct {
	Name   string   `json:"n"`
	Events []*Event `json:"e"`
}

// Count returns the number of events contained in this
// timing.
func (t *Timing) Count() int {
	return len(t.Events)
}

// Total returns the total elapsed time for all the events
// in this Timing.
func (t *Timing) Total() time.Duration {
	total := time.Duration(0)
	for _, v := range t.Events {
		total += v.Elapsed()
	}
	return total
}
