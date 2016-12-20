// +build appengine

package tasks

import (
	"sync"

	"gnd.la/app"
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
	app.Signals.WillPrepare.Listen(func(a *App) {
		a.Handle("/gondola-run-tasks", gondolaRunTasksHandler)
	})
}
