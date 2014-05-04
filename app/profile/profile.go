package profile

import (
	"fmt"
	"sync"
	"time"

	"gnd.la/log"
)

const On = profileIsOn

// Timed represents an ongoing timed event.
// Use Start or Startf to start a timed
// event.
type Timed struct {
	name    string
	started time.Time
	ended   time.Time
	autoend bool
	notes   []*Note
}

// Note adds a note regarding this timed event (e.g. the
// SQL query that was executed, the URL that was fetched, etc...).
func (t *Timed) Note(title string, text string) *Timed {
	t.notes = append(t.notes, &Note{Title: title, Text: text})
	return t
}

// Notef works like Note(), but accepts a format string.
func (t *Timed) Notef(title string, format string, args ...interface{}) *Timed {
	t.notes = append(t.notes, &Note{Title: title, Text: fmt.Sprintf(format, args...)})
	return t
}

// End ends the timed event.
func (t *Timed) End() {
	t.ended = time.Now()
}

// AutoEnd causes the timed event to be ended when
// the timings are requested.
func (e *Timed) AutoEnd() {
	e.autoend = true
}

type context struct {
	sync.Mutex
	events []*Timed
}

var contexts struct {
	sync.RWMutex
	data map[int32]*context
}

func currentEvent() *Timed {
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

// ID returns the profiling ID for the current goroutine.
func ID() int {
	return int(goroutineId())
}

// Begin enables profiling for the current goroutine.
// This function is idempotent. Any goroutine which
// calls Begin must also call End to avoid leaking
// resources.
func Begin() {
	gid := goroutineId()
	if gid >= 0 {
		contexts.Lock()
		if _, ok := contexts.data[gid]; !ok {
			contexts.data[gid] = &context{}
		}
		contexts.Unlock()
	}
}

// End removes any profiling data for this goroutine. It must
// called before the goroutine ends for each goroutine which
// called Begin(). If parent is non-zero, the events in the
// ending goroutine are added to the goroutine with the given ID.
func End(parent int) {
	gid := goroutineId()
	contexts.Lock()
	if parent > 0 {
		cur := contexts.data[gid]
		if cur != nil {
			p := contexts.data[int32(parent)]
			if p != nil {
				// We're assumming well behaved clients, which
				// won't call End() with arguments creating cycles,
				// so there should be no risk of deadlock here.
				p.Lock()
				cur.Lock()
				p.events = append(p.events, cur.events...)
				cur.Unlock()
				p.Unlock()
			}
		}
	}
	delete(contexts.data, gid)
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

// Start starts a timed event. Use Timed.End to terminate the
// event or Timed.AutoEnd to finish it when the request finishes
// processing. Note that if profiling is not enabled for the current
// goroutine, this function does nothing and returns an empty event.
func Start(name string) *Timed {
	contexts.RLock()
	ctx := contexts.data[goroutineId()]
	contexts.RUnlock()
	if ctx == nil {
		return &Timed{}
	}
	ev := &Timed{name: name, started: time.Now()}
	ctx.Lock()
	ctx.events = append(ctx.events, ev)
	ctx.Unlock()
	return ev
}

// Startf is a shorthand function for calling Start and then
// Timed.Notef on the resulting Ev.
func Startf(name string, title string, format string, args ...interface{}) *Timed {
	e := Start(name)
	e.Notef(title, format, args...)
	return e
}

// HasEvent returns true iff there's current an ongoing
// timed event for the current goroutine.
func HasEvent() bool {
	return currentEvent() != nil
}

// Notef adds a note to the current Timed, as started by Start
// or Startf.
func Notef(title string, format string, args ...interface{}) {
	ev := currentEvent()
	if ev == nil {
		log.Warningln("can't note, no ongoing event")
		return
	}
	ev.Notef(title, format, args...)
}

// Profile is a shorthand function for calling Start(),
// executing f and the calling Timed.End() on the resulting
// Timed.
func Profile(f func(), name string) {
	ev := Start(name)
	f()
	ev.End()
}

// Profilef is a shorthand function for calling Startf(),
// executing f and the calling Timed.End() on the resulting
// Timed.
func Profilef(f func(), title string, name string, format string, args ...interface{}) {
	ev := Startf(name, title, format, args...)
	f()
	ev.End()
}

// Timings returns the available timings for the current
// goroutine. Note that calling Timings will end any events
// which have been set to automatically end (with Timed.AutoEnd)
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
