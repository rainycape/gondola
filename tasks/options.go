package tasks

type options struct {
	Name         string
	MaxInstances int
	RunOnListen  bool
}

func prepareOptions(opts []optsFunc) options {
	var o options
	for _, f := range opts {
		o = f(o)
	}
	return o
}

type optsFunc func(options) options

// Name sets the task name, used for checking the number
// of instances running. If the task name is not provided, it's
// derived from the function. Two tasks with the same name are
// considered as equal, even if their functions are different.
func Name(name string) optsFunc {
	return func(o options) options {
		o.Name = name
		return o
	}
}

// MaxInstances indicates the maximum number of instances of
// this function that can be simultaneously running. If zero
// or unspecified, there is no limit.
func MaxInstances(n int) optsFunc {
	return func(o options) options {
		o.MaxInstances = n
		return o
	}
}

// RunOnListen makes the task run (in its own goroutine) as soon as the app starts listening
// rather than waiting until its scheduled interval for the first run. Note that on App Engine,
// the task will be started when the first cron request comes in.
func RunOnListen() optsFunc {
	return func(o options) options {
		o.RunOnListen = true
		return o
	}
}
