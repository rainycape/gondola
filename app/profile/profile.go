// +build profile

package profile

import (
	"fmt"
	"gnd.la/log"
	"sync"
	"time"
)

const On = true

type Ev struct {
	name    string
	started time.Time
	ended   time.Time
	autoend bool
	notes   []string
}

func (e *Ev) Note(format string, args ...interface{}) {
	e.notes = append(e.notes, fmt.Sprintf(format, args...))
}

func (e *Ev) End() {
	e.ended = time.Now()
}

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

func Start(name string) *Ev {
	contexts.RLock()
	ctx := contexts.data[goroutineId()]
	contexts.RUnlock()
	if ctx == nil {
		ctx = &context{}
		contexts.Lock()
		contexts.data[goroutineId()] = ctx
		contexts.Unlock()
	}
	ev := &Ev{name: name, started: time.Now()}
	ctx.Lock()
	ctx.events = append(ctx.events, ev)
	ctx.Unlock()
	return ev
}

func Startf(name string, format string, args ...interface{}) *Ev {
	e := Start(name)
	e.Note(format, args...)
	return e
}

func HasEvent() bool {
	return currentEvent() != nil
}

func Note(format string, args ...interface{}) {
	ev := currentEvent()
	if ev == nil {
		log.Warningln("can't note, no ongoing event")
	} else {
		ev.Note(format, args...)
	}
}

func End() {
	contexts.Lock()
	delete(contexts.data, goroutineId())
	contexts.Unlock()
}

func Profile(f func(), name string) {
	ev := Start(name)
	f()
	ev.End()
}

func Profilef(f func(), name string, format string, args ...interface{}) {
	ev := Startf(name, format, args...)
	f()
	ev.End()
}

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
