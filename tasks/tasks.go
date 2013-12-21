// Package tasks provides functions for scheduling
// periodic tasks (e.g. background jobs).
package tasks

import (
	"bytes"
	"fmt"
	"gnd.la/log"
	"gnd.la/mux"
	"gnd.la/signal"
	"gnd.la/util/internal/runtimeutil"
	"reflect"
	"runtime"
	"sync"
	"time"
)

var running struct {
	sync.Mutex
	tasks map[uintptr]int
}

var options struct {
	sync.RWMutex
	options map[uintptr]*Options
}

// Task represent a scheduled task.
type Task struct {
	Mux      *mux.Mux
	Handler  mux.Handler
	Interval time.Duration
	Options  *Options
	ticker   *time.Ticker
}

// Stop de-schedules the task. After stopping the task, it
// won't be started again but if it's currently running, it will
// be completed.
func (t *Task) Stop() {
	if t.ticker != nil {
		t.ticker.Stop()
	}
}

// Options are used to specify task options when registering them.
type Options struct {
	// If true, the task is not run at the time of scheduling it.
	// Id est, its first run will take place after the specified
	// interval.
	NotNow bool
	// MaxInstances indicates the maximum number of instances of
	// this function that can be simultaneously running. If zero,
	// there is no limit.
	MaxInstances int
}

func taskKey(task mux.Handler) uintptr {
	return reflect.ValueOf(task).Pointer()
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
	if opts != nil && opts.MaxInstances > 0 {
		k := taskKey(task)
		running.Lock()
		defer running.Unlock()
		c := running.tasks[k]
		if c > 1 {
			running.tasks[k] = c - 1
		} else {
			delete(running.tasks, k)
		}
	}
}

func tryRunTask(task mux.Handler, opts *Options) bool {
	if opts != nil && opts.MaxInstances > 0 {
		k := taskKey(task)
		running.Lock()
		defer running.Unlock()
		c := running.tasks[k]
		if c >= opts.MaxInstances {
			name := runtimeutil.FuncName(task)
			log.Warningf("Not starting task %s because it's already running %d instances", name, c)
			return false
		}
		if running.tasks == nil {
			running.tasks = make(map[uintptr]int)
		}
		running.tasks[k] = c + 1
	}
	return true
}

func taskOptions(task mux.Handler) *Options {
	options.RLock()
	defer options.RUnlock()
	return options.options[taskKey(task)]
}

func executeTask(m *mux.Mux, task mux.Handler, opts *Options) {
	if !tryRunTask(task, opts) {
		return
	}
	name := runtimeutil.FuncName(task)
	ctx := m.NewContext(contextProvider(0))
	defer m.CloseContext(ctx)
	now := time.Now()
	log.Debugf("Starting task %s at %v", name, now)
	defer afterTask(task, name, now, opts)
	task(ctx)
}

// Schedule schedules a task to be run at the given interval. Unless the NotNow option
// is specified, the task is also run (in a goroutine) just after being scheduled,
// rather than waiting until interval for the first run.
func Schedule(m *mux.Mux, interval time.Duration, opts *Options, task mux.Handler) *Task {
	ticker := time.NewTicker(interval)
	if opts != nil {
		options.Lock()
		if options.options == nil {
			options.options = make(map[uintptr]*Options)
		}
		options.options[taskKey(task)] = opts
		options.Unlock()
	}
	t := &Task{Mux: m, Handler: task, Interval: interval, Options: opts, ticker: ticker}
	signal.Emit(signal.TASKS_WILL_SCHEDULE_TASK, t)
	go func() {
		if opts == nil || !opts.NotNow {
			executeTask(m, task, opts)
		}
		for _ = range ticker.C {
			executeTask(m, task, opts)
		}
	}()
	return t
}

// Run starts the given task, unless it has been previously
// registered with Options which prevent from running it right
// now (e.g. it was scheduled with MaxInstances = 2 and
// there are already 2 instances running).
func Run(m *mux.Mux, task mux.Handler) {
	opts := taskOptions(task)
	executeTask(m, task, opts)
}
