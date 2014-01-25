package signal

const (
	// APP_WILL_LISTEN is emitted just before a *gnd.la/app.App will
	// start listening. The object is the App.
	APP_WILL_LISTEN = "APP_WILL_LISTEN"
	// TASKS_WILL_SCHEDULE_TASK is emitted just before scheduling a task.
	// The object is a *gnd.la/tasks.Task
	TASKS_WILL_SCHEDULE_TASK = "TASKS_WILL_SCHEDULE_TASK"
	// ORM_WILL_INITIALIZE is emitted just before a gnd.la/orm.Orm is
	// initialized. The object is a *gnd.la/orm.Orm.
	ORM_WILL_INITIALIZE = "ORM_WILL_INITIALIZE"
)
