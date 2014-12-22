package commands

type contextProvider struct {
	args        []string
	params      []string
	paramValues map[string]string
}

func (c *contextProvider) Count() int {
	return len(c.args)
}

func (c *contextProvider) Arg(i int) string {
	if i < len(c.args) {
		return c.args[i]
	}
	return ""
}

func (c *contextProvider) Param(name string) string {
	return c.paramValues[name]
}

func (c *contextProvider) Params() []string {
	return c.params
}
