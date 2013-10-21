package assets

import (
	"fmt"
	"io/ioutil"
	"strings"
)

type Attributes map[string]string

func (a Attributes) String() string {
	var attrs []string
	for k, v := range map[string]string(a) {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", k, strings.Replace(v, "\"", "\\\"", -1)))
	}
	return strings.Join(attrs, " ")
}

type Common struct {
	Manager   Manager
	Name      string
	Condition *Condition
}

func (c *Common) AssetName() string {
	return c.Name
}

func (c *Common) AssetCondition() *Condition {
	return c.Condition
}

func (c *Common) Code() (string, error) {
	f, _, err := c.Manager.Load(c.Name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseCommon(m Manager, names []string, options Options) ([]*Common, error) {
	var cond *Condition
	if ifopt := options["if"]; ifopt != "" {
		var err error
		cond, err = ParseCondition(ifopt)
		if err != nil {
			return nil, err
		}
	}
	common := make([]*Common, len(names))
	for ii, v := range names {
		common[ii] = &Common{
			Manager:   m,
			Name:      v,
			Condition: cond,
		}
	}
	return common, nil
}
