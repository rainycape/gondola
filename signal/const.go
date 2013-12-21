package signal

const (
	// CONFIGURED is emitted after the configuration has been parsed.
	// The object is the configuration itself.
	CONFIGURED = "CONFIGURED"
	// MUX_WILL_LISTEN is emitted just before a *gnd.la/mux/Mux will
	// started listening. The
	// object is the Mux.
	MUX_WILL_LISTEN = "MUX_WILL_LISTEN"
	// TASKS_WILL_SCHEDULE_TASK is emitted just before scheduling a task.
	// The object is a *gnd.la/tasks.Task
	TASKS_WILL_SCHEDULE_TASK = "TASKS_WILL_SCHEDULE_TASK"
)
