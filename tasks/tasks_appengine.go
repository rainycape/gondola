// +build appengine

package tasks

import (
	"sync"

	"gnd.la/app"
	"gnd.la/log"
	"gnd.la/signal"
)

var pendingTasks struct {
	sync.RWMutex
	tasks []*Task
}

func startTask(task *Task) error {
	pendingTasks.Lock()
	pendingTasks.tasks = append(pendingTasks.tasks, task)
	pendingTasks.Unlock()
	return nil
}

func gondolaRunTasksHandler(ctx *app.Context) {
	if ctx.GetHeader("X-Appengine-Cron") != "true" {
		ctx.Forbidden("")
		return
	}
	pendingTasks.Lock()
	for _, v := range pendingTasks.tasks {
		task := v
		ctx.Go(func(c *app.Context) {
			if _, err := executeTask(c, task); err != nil {
				log.Error(err)
			}
		})
	}
	pendingTasks.tasks = nil
	pendingTasks.Unlock()
	ctx.Wait()
}

func init() {
	signal.Register(app.WILL_PREPARE, func(_ string, obj interface{}) {
		a := obj.(*app.App)
		a.Handle("/gondola-run-tasks", gondolaRunTasksHandler)
	})
}
