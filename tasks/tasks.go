// Package tasks provides functions for scheduling
// periodic tasks (e.g. background jobs).
package tasks

import (
	"bytes"
	"fmt"
	"gnd.la/log"
	"gnd.la/mux"
	"gnd.la/runtimeutil"
	"runtime"
	"time"
)

// Task represent a scheduled task.
type Task time.Ticker

// Stop stops the task. After stopping the task, it won't be started
// again but if it's currently running, it will be completed.
func (t *Task) Stop() {
	(*time.Ticker)(t).Stop()
}

func afterTask(name string, started time.Time) {
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
}

func executeTask(m *mux.Mux, task mux.Handler) {
	name := runtimeutil.FuncName(task)
	ctx := m.NewContext(contextProvider(0))
	defer m.CloseContext(ctx)
	now := time.Now()
	log.Debugf("Starting task %s at %v", name, now)
	defer afterTask(name, now)
	task(ctx)
}

// Schedule schedules a task to be run at the given interval. If the now parameter
// is true, the task is also executed right now (in a goroutine), rather than waiting
// until interval for the first run.
func Schedule(m *mux.Mux, interval time.Duration, now bool, task mux.Handler) *Task {
	ticker := time.NewTicker(interval)
	go func() {
		if now {
			executeTask(m, task)
		}
		for _ = range ticker.C {
			executeTask(m, task)
		}
	}()
	return (*Task)(ticker)
}
