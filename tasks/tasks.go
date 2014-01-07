// Package tasks provides functions for scheduling
// periodic tasks (e.g. background jobs).
package tasks

import (
	"bytes"
	"errors"
	"fmt"
	"gnd.la/app"
	"gnd.la/log"
	"gnd.la/signal"
	"gnd.la/util/internal/runtimeutil"
	"reflect"
	"sync"
	"time"
)

var running struct {
	sync.Mutex
	tasks map[*Task]int
}

var registered struct {
	sync.RWMutex
	tasks map[string]*Task
}

// Task represent a scheduled task.
type Task struct {
	App      *app.App
	Handler  app.Handler
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
		t.ticker = nil
	}
}

func (t *Task) Resume(now bool) {
	t.Stop()
	t.ticker = time.NewTicker(t.Interval)
	go t.execute(now)
}

// Name returns the task name.
func (t *Task) Name() string {
	if t.Options != nil && t.Options.Name != "" {
		return t.Options.Name
	}
	return runtimeutil.FuncName(t.Handler)
}

// Delete stops the task by calling t.Stop() and then removes
// it from the internal task register.
func (t *Task) Delete() {
	registered.Lock()
	defer registered.Unlock()
	t.deleteLocked()
}

func (t *Task) deleteLocked() {
	t.Stop()
	delete(registered.tasks, t.Name())
}

func (t *Task) execute(now bool) {
	if now {
		t.executeTask()
	}
	for _ = range t.ticker.C {
		t.executeTask()
	}
}

func (t *Task) executeTask() {
	if _, err := executeTask(t); err != nil {
		log.Error(err)
	}
}

// Options are used to specify task options when registering them.
type Options struct {
	// Name indicates the task name, used for checking the number
	// of instances running. If the task name is not provided, it's
	// derived from the function. Two tasks with the same name are
	// considered as equal, even if their functions are different.
	Name string
	// MaxInstances indicates the maximum number of instances of
	// this function that can be simultaneously running. If zero,
	// there is no limit.
	MaxInstances int
}

func afterTask(task *Task, started time.Time, terr *error) {
	name := task.Name()
	if err := recover(); err != nil {
		skip, stackSkip, _, _ := runtimeutil.GetPanic()
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "Panic executing task %s: %v\n", name, err)
		stack := runtimeutil.FormatStack(stackSkip)
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
		*terr = errors.New(buf.String())
	}
	end := time.Now()
	log.Debugf("Finished task %s at %v (took %v)", name, end, end.Sub(started))
	running.Lock()
	defer running.Unlock()
	c := running.tasks[task]
	if c > 1 {
		running.tasks[task] = c - 1
	} else {
		delete(running.tasks, task)
	}
}

func canRunTask(task *Task) error {
	running.Lock()
	defer running.Unlock()
	c := running.tasks[task]
	if task.Options != nil && task.Options.MaxInstances > 0 {
		if c >= task.Options.MaxInstances {
			return fmt.Errorf("not starting task %s because it's already running %d instances", task.Name(), c)
		}
	}
	if running.tasks == nil {
		running.tasks = make(map[*Task]int)
	}
	running.tasks[task] = c + 1
	return nil
}

func executeTask(task *Task) (ran bool, err error) {
	if err = canRunTask(task); err != nil {
		return
	}
	ctx := task.App.NewContext(contextProvider(0))
	defer task.App.CloseContext(ctx)
	started := time.Now()
	log.Debugf("Starting task %s at %v", task.Name(), started)
	ran = true
	defer afterTask(task, started, &err)
	task.Handler(ctx)
	return
}

// Register registers a new task that might be run with Run, but
// without scheduling it. If there was previously another task
// registered with the same name, it will be deleted.
func Register(m *app.App, task app.Handler, opts *Options) *Task {
	t := &Task{App: m, Handler: task, Options: opts}
	registered.Lock()
	defer registered.Unlock()
	if registered.tasks == nil {
		registered.tasks = make(map[string]*Task)
	}
	name := t.Name()
	if prev := registered.tasks[name]; prev != nil {
		log.Debugf("There's already a task registered as %s, deleting it", name)
		prev.deleteLocked()
	}
	registered.tasks[name] = t
	return t
}

// Schedule registers and schedules a task to be run at the given
// interval. If interval is 0, the task is only registered, but not
// scheduled. The now argument indicates if the task should also run
// right now (in a goroutine) rather than waiting until interval for
// the first run. Schedule returns a Task instance, which might be
// used to stop, resume or delete a it.
func Schedule(m *app.App, task app.Handler, opts *Options, interval time.Duration, now bool) *Task {
	t := Register(m, task, opts)
	t.Interval = interval
	signal.Emit(signal.TASKS_WILL_SCHEDULE_TASK, t)
	go t.Resume(now)
	return t
}

// Run starts the given task identifier by it's name, unless
// it has been previously registered with Options which
// prevent from running it right now (e.g. it was registered
// with MaxInstances = 2 and there are already 2 instances running).
// The first return argument indicates if the task was executed, while
// the second includes any errors which happened while running the task.
func Run(name string) (bool, error) {
	registered.RLock()
	task := registered.tasks[name]
	registered.RUnlock()
	if task == nil {
		return false, fmt.Errorf("there's no task registered with the name %q", name)
	}
	return executeTask(task)
}

// RunHandler starts the given task identifier by it's handler. The same
// restrictions in Run() apply to this function.
// Return values are the same as Run().
func RunHandler(handler app.Handler) (bool, error) {
	var task *Task
	p := reflect.ValueOf(handler).Pointer()
	registered.RLock()
	for _, v := range registered.tasks {
		if reflect.ValueOf(v.Handler).Pointer() == p {
			task = v
			break
		}
	}
	registered.RUnlock()
	if task == nil {
		return false, fmt.Errorf("there's no task registered with the handler %s", runtimeutil.FuncName(handler))
	}
	return executeTask(task)
}

// Execute runs the given handler in a task context. If the handler fails
// with a panic, it will be returned in the error return value.
func Execute(a *app.App, handler app.Handler) error {
	t := &Task{App: a, Handler: handler}
	_, err := executeTask(t)
	return err
}
