package tasks

// context provider for tasks. Since the tasks
// receive no parameters, the provider is just
// a dummy one which always returns zero/empty.
type contextProvider struct{}

func (c contextProvider) Count() int {
	return 0
}

func (c contextProvider) Arg(i int) (string, bool) {
	return "", false
}

func (c contextProvider) Param(name string) (string, bool) {
	return "", false
}

func (c contextProvider) ParamNames() []string {
	return nil
}
