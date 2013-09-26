package tasks

// context provider for tasks. Since the tasks
// receive no parameters, the provider is just
// a dummy one which always returns zero/empty.
type contextProvider byte

func (c contextProvider) Count() int {
	return 0
}

func (c contextProvider) Arg(i int) string {
	return ""
}

func (c contextProvider) Param(name string) string {
	return ""
}
