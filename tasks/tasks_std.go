// +build !appengine

package tasks

func startTask(task *Task) error {
	ctx := task.App.NewContext(contextProvider(0))
	defer task.App.CloseContext(ctx)
	_, err := executeTask(ctx, task)
	return err
}
