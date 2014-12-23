// +build !appengine

package tasks

func (t *Task) executeTask() {
	ctx := t.App.NewContext(contextProvider{})
	defer t.App.CloseContext(ctx)
	_, err := executeTask(ctx, t)
	if err != nil {
		ctx.Logger().Error(err)
	}
}
