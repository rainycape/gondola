package commands

type contextProvider struct {
	args        []string
	params      []string
	paramValues map[string]string
}

func (c *contextProvider) Count() int {
	return len(c.args)
}

func (c *contextProvider) Arg(i int) (string, bool) {
	if i < len(c.args) {
		return c.args[i], true
	}
	return "", false
}

func (c *contextProvider) Param(name string) (string, bool) {
	val, found := c.paramValues[name]
	return val, found
}

func (c *contextProvider) ParamNames() []string {
	return c.params
}
