// +build appengine

package tasks

import (
	"sync"

	"gnd.la/app"
	"gnd.la/signal"
)

var pendingTasks struct {
	sync.RWMutex
	tasks []*Task
}

func (t *Task) executeTask() {
	pendingTasks.Lock()
	pendingTasks.tasks = append(pendingTasks.tasks, t)
	pendingTasks.Unlock()
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
				ctx.Logger().Error(err)
			}
		})
	}
	pendingTasks.tasks = nil
	pendingTasks.Unlock()
	ctx.Wait()
}

func init() {
	signal.Listen(app.WILL_PREPARE, func(_ string, obj interface{}) {
		a := obj.(*app.App)
		a.Handle("/gondola-run-tasks", gondolaRunTasksHandler)
	})
}
