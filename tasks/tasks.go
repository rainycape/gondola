// Package tasks provides functions for scheduling
// periodic tasks (e.g. background jobs).
package tasks

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"gnd.la/app"
	"gnd.la/internal/runtimeutil"
	"gnd.la/signal"
)

var running struct {
	sync.Mutex
	tasks map[*Task]int
}

var registered struct {
	sync.RWMutex
	tasks map[string]*Task
}

var onListenTasks struct {
	sync.RWMutex
	tasks []*Task
}

// Task represent a scheduled task.
type Task struct {
	App      *app.App
	Handler  app.Handler
	Interval time.Duration
	Options  *Options
	ticker   *time.Ticker
	stop     chan struct{}
	stopped  chan struct{}
}

// Stop de-schedules the task. After stopping the task, it
// won't be started again but if it's currently running, it will
// be completed.
func (t *Task) Stop() {
	if t.stop != nil {
		t.stop <- struct{}{}
		<-t.stopped
		close(t.stopped)
		t.stopped = nil
	}
}

func (t *Task) Resume(now bool) {
	t.Stop()
	t.ticker = time.NewTicker(t.Interval)
	t.stop = make(chan struct{}, 1)
	t.stopped = make(chan struct{}, 1)
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
	for {
		c := t.ticker.C
		select {
		case <-c:
			go t.executeTask()
		case <-t.stop:
			close(t.stop)
			t.stop = nil
			t.ticker.Stop()
			t.ticker = nil
			t.stopped <- struct{}{}
			return
		}
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

func afterTask(ctx *app.Context, task *Task, started time.Time, terr *error) {
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
	running.Lock()
	defer running.Unlock()
	c := running.tasks[task] - 1
	if c > 0 {
		running.tasks[task] = c
	} else {
		delete(running.tasks, task)
	}
	ctx.Logger().Infof("Finished task %s (%d instances now running) at %v (took %v)", name, c, end, end.Sub(started))
}

func numberOfInstances(task *Task) (int, error) {
	running.Lock()
	defer running.Unlock()
	c := running.tasks[task]
	if task.Options != nil && task.Options.MaxInstances > 0 {
		if c >= task.Options.MaxInstances {
			return 0, fmt.Errorf("not starting task %s because it's already running %d instances", task.Name(), c)
		}
	}
	if running.tasks == nil {
		running.tasks = make(map[*Task]int)
	}
	c++
	running.tasks[task] = c
	return c, nil
}

func executeTask(ctx *app.Context, task *Task) (ran bool, err error) {
	var n int
	if n, err = numberOfInstances(task); err != nil {
		return
	}
	started := time.Now()
	ctx.Logger().Infof("Starting task %s (%d instances now running) at %v", task.Name(), n, started)
	ran = true
	defer afterTask(ctx, task, started, &err)
	task.Handler(ctx)
	return
}

// Register registers a new task that might be run with Run, but
// without scheduling it. If there was previously another task
// registered with the same name, it will panic (use Task.Delete
// previously to remove it).
func Register(m *app.App, task app.Handler, opts *Options) *Task {
	t := &Task{App: m, Handler: task, Options: opts}
	registered.Lock()
	defer registered.Unlock()
	if registered.tasks == nil {
		registered.tasks = make(map[string]*Task)
	}
	name := t.Name()
	if prev := registered.tasks[name]; prev != nil {
		panic(fmt.Errorf("there's already a task registered as %s", name))
	}
	registered.tasks[name] = t
	return t
}

// Schedule registers and schedules a task to be run at the given
// interval. If interval is 0, the task is only registered, but not
// scheduled. The onListen argument indicates if the task should also run
// (in its own goroutine) as soon as the app starts listening rather than
// waiting until interval for the first run. Note that on App Engine, the task
// will be started when the first cron request comes in (these are usually scheduled
// to run once a minute).
//
// Schedule returns a Task instance, which might be used to stop, resume or delete a it.
func Schedule(m *app.App, task app.Handler, opts *Options, interval time.Duration, onListen bool) *Task {
	t := Register(m, task, opts)
	t.Interval = interval
	go t.Resume(false)
	if onListen {
		onListenTasks.Lock()
		onListenTasks.tasks = append(onListenTasks.tasks, t)
		onListenTasks.Unlock()
	}
	return t
}

// Run starts the given task identifier by it's name, unless
// it has been previously registered with Options which
// prevent from running it right now (e.g. it was registered
// with MaxInstances = 2 and there are already 2 instances running).
// The first return argument indicates if the task was executed, while
// the second includes any errors which happened while running the task.
func Run(ctx *app.Context, name string) (bool, error) {
	registered.RLock()
	task := registered.tasks[name]
	registered.RUnlock()
	if task == nil {
		return false, fmt.Errorf("there's no task registered with the name %q", name)
	}
	return executeTask(ctx, task)
}

// RunHandler starts the given task identifier by it's handler. The same
// restrictions in Run() apply to this function.
// Return values are the same as Run().
func RunHandler(ctx *app.Context, handler app.Handler) (bool, error) {
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
	return executeTask(ctx, task)
}

// Execute runs the given handler in a task context. If the handler fails
// with a panic, it will be returned in the error return value.
func Execute(ctx *app.Context, handler app.Handler) error {
	t := &Task{App: ctx.App(), Handler: handler}
	_, err := executeTask(ctx, t)
	return err
}

func init() {
	// Admin commands are executed on WILL_PREPARE so we
	// won't reach this point if there's an admin command
	// provided in the cmdline.
	signal.Listen(app.DID_PREPARE, func(_ string, obj interface{}) {
		a := obj.(*app.App)
		onListenTasks.Lock()
		var pending []*Task
		for _, v := range onListenTasks.tasks {
			if v.App == a {
				go v.executeTask()
			} else {
				pending = append(pending, v)
			}
		}
		onListenTasks.tasks = pending
		onListenTasks.Unlock()
	})
}
