// Package tasks provides functions for scheduling
// periodic tasks (e.g. background jobs).
package tasks

import (
	"bytes"
	"fmt"
	"gnd.la/log"
	"gnd.la/mux"
	"gnd.la/util/runtimeutil"
	"reflect"
	"runtime"
	"sync"
	"time"
)

var running struct {
	sync.Mutex
	tasks map[uintptr]bool
}

// Task represent a scheduled task.
type Task time.Ticker

// Stop stops the task. After stopping the task, it won't be started
// again but if it's currently running, it will be completed.
func (t *Task) Stop() {
	(*time.Ticker)(t).Stop()
}

// Options are used to specify task options when registering them.
type Options struct {
	// If true, the task is not run at the time of scheduling it.
	// Id est, its first run will take place after the specified
	// interval.
	NotNow bool
	// Indicates that this task should not be started if there's
	// already an instance of this task running.
	Unique bool
}

func afterTask(task mux.Handler, name string, started time.Time, opts *Options) {
	if err := recover(); err != nil {
		skip := 2
		if _, ok := err.(runtime.Error); ok {
			// When err is a runtime.Error, there are two
			// additional stack frames inside the runtime
			// which are the ones finally calling panic()
			skip += 2
		}
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("Panic executing task %s: ", name))
		buf.WriteString(fmt.Sprintf("%v", err))
		buf.WriteByte('\n')
		stack := runtimeutil.FormatStack(2)
		location, code := runtimeutil.FormatCaller(skip, 5, true, true)
		if location != "" {
			buf.WriteString("\n At ")
			buf.WriteString(location)
			if code != "" {
				buf.WriteByte('\n')
				buf.WriteString(code)
				buf.WriteByte('\n')
			}
		}
		if stack != "" {
			buf.WriteString("\nStack:\n")
			buf.WriteString(stack)
		}
		log.Error(buf.String())
	}
	end := time.Now()
	log.Debugf("Finished task %s at %v (took %v)", name, end, end.Sub(started))
	if opts != nil && opts.Unique {
		running.Lock()
		delete(running.tasks, reflect.ValueOf(task).Pointer())
		running.Unlock()
	}
}

func executeTask(m *mux.Mux, task mux.Handler, opts *Options) {
	name := runtimeutil.FuncName(task)
	ctx := m.NewContext(contextProvider(0))
	defer m.CloseContext(ctx)
	now := time.Now()
	if opts != nil && opts.Unique {
		k := reflect.ValueOf(task).Pointer()
		running.Lock()
		if running.tasks[k] {
			log.Errorf("Not starting task %s because it's already running", name)
			running.Unlock()
			return
		}
		if running.tasks == nil {
			running.tasks = make(map[uintptr]bool)
		}
		running.tasks[k] = true
		running.Unlock()
	}
	log.Debugf("Starting task %s at %v", name, now)
	defer afterTask(task, name, now, opts)
	task(ctx)
}

// Schedule schedules a task to be run at the given interval. Unless the NotNow option
// is specified, the task is also run (in a goroutine) just after being scheduled,
// rather than waiting until interval for the first run.
func Schedule(m *mux.Mux, interval time.Duration, opts *Options, task mux.Handler) *Task {
	ticker := time.NewTicker(interval)
	go func() {
		if opts == nil || !opts.NotNow {
			executeTask(m, task, opts)
		}
		for _ = range ticker.C {
			executeTask(m, task, opts)
		}
	}()
	return (*Task)(ticker)
}
