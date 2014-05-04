// +build profile

package profile

import (
	"fmt"
	"sync"
	"time"

	"gnd.la/log"
)

const On = true

// Ev represents an ongoing timed event.
// Use Start or Startf to start a timed
// event.
type Ev struct {
	name    string
	started time.Time
	ended   time.Time
	autoend bool
	notes   []string
}

// Note adds a note regarding this timed event (e.g. the
// SQL query that was executed, the URL that was fetched, etc...).
func (e *Ev) Note(format string, args ...interface{}) {
	e.notes = append(e.notes, fmt.Sprintf(format, args...))
}

// End ends the timed event.
func (e *Ev) End() {
	e.ended = time.Now()
}

// AutoEnd causes the timed event to be ended when
// the request is served.
func (e *Ev) AutoEnd() {
	e.autoend = true
}

type context struct {
	sync.Mutex
	events []*Ev
}

var contexts struct {
	sync.RWMutex
	data map[int32]*context
}

func goroutineId() int32

func currentEvent() *Ev {
	contexts.RLock()
	ctx := contexts.data[goroutineId()]
	contexts.RUnlock()
	if ctx != nil {
		ctx.Lock()
		defer ctx.Unlock()
		if len(ctx.events) > 0 {
			return ctx.events[len(ctx.events)-1]
		}
	}
	return nil
}

// Begin enables profiling for the current goroutine.
// This function is idempotent. Any goroutine which
// calls Begin must also call End to avoid leaking
// resources.
func Begin() {
	gid := goroutineId()
	contexts.Lock()
	if _, ok := contexts.data[gid]; !ok {
		contexts.data[gid] = &context{}
	}
	contexts.Unlock()
}

// End removes any profiling data for this goroutine. It must
// called before the goroutine ends for each goroutine which
// called Begin().
func End() {
	contexts.Lock()
	delete(contexts.data, goroutineId())
	contexts.Unlock()
}

// Profiling returns wheter profiling has been enabled for
// this goroutine.
func Profiling() bool {
	contexts.RLock()
	_, ok := contexts.data[goroutineId()]
	contexts.RUnlock()
	return ok
}

// Start starts a timed event. Use Ev.End to terminate the
// event or Ev.AutoEnd to finish it when the request finishes
// processing. Note that if profiling is not enabled for the current
// goroutine, this function does nothing and returns an empty event.
func Start(name string) *Ev {
	contexts.RLock()
	ctx := contexts.data[goroutineId()]
	contexts.RUnlock()
	if ctx == nil {
		return &Ev{}
	}
	ev := &Ev{name: name, started: time.Now()}
	ctx.Lock()
	ctx.events = append(ctx.events, ev)
	ctx.Unlock()
	return ev
}

// Startf is a shorthand function for calling Start and then
// Ev.Note on the resulting Ev.
func Startf(name string, format string, args ...interface{}) *Ev {
	e := Start(name)
	e.Note(format, args...)
	return e
}

// HasEvent returns true iff there's current an ongoing
// timed event for the current goroutine.
func HasEvent() bool {
	return currentEvent() != nil
}

// Note adds a note to the current Ev, as started by Start
// or Startf.
func Note(format string, args ...interface{}) {
	ev := currentEvent()
	if ev == nil {
		log.Warningln("can't note, no ongoing event")
	} else {
		ev.Note(format, args...)
	}
}

// Profile is a shorthand function for calling Start(),
// executing f and the calling Ev.End() on the resulting
// Ev.
func Profile(f func(), name string) {
	ev := Start(name)
	f()
	ev.End()
}

// Profilef is a shorthand function for calling Startf(),
// executing f and the calling Ev.End() on the resulting
// Ev.
func Profilef(f func(), name string, format string, args ...interface{}) {
	ev := Startf(name, format, args...)
	f()
	ev.End()
}

// Timings returns the available timings for the current
// goroutine. Note that calling Timings will end any events
// which have been set to automatically end (with Ev.AutoEnd)
// so this function should only be called at the end of the
// request lifecycle.
func Timings() []*Timing {
	var timings map[string]*Timing
	contexts.RLock()
	ctx := contexts.data[goroutineId()]
	if ctx != nil {
		timings = make(map[string]*Timing)
		ctx.Lock()
		for _, v := range ctx.events {
			if v.ended.IsZero() {
				if v.autoend {
					v.End()
				} else {
					if len(v.notes) > 0 {
						log.Warningf("unfinished %q event (%s)", v.name, v.notes)
					} else {
						log.Warningf("unfinished %q event", v.name)
					}
					continue
				}
			}
			timing := timings[v.name]
			if timing == nil {
				timing = &Timing{Name: v.name}
				timings[v.name] = timing
			}
			timing.Events = append(timing.Events, &Event{v.started, v.ended, v.notes})
		}
		ctx.Unlock()
	}
	contexts.RUnlock()
	ret := make([]*Timing, 0, len(timings))
	for _, v := range timings {
		ret = append(ret, v)
	}
	return ret
}

func init() {
	contexts.data = make(map[int32]*context)
}
