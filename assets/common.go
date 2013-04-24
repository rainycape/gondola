package assets

import (
	"fmt"
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

type CommonAsset struct {
	Manager          Manager
	name             string
	condition        Condition
	conditionVersion int
}

func (c *CommonAsset) Name() string {
	return c.name
}

func (c *CommonAsset) Condition() Condition {
	return c.condition
}

func (c *CommonAsset) ConditionVersion() int {
	return c.conditionVersion
}

func ParseCommonAssets(m Manager, names []string, options Options) ([]*CommonAsset, error) {
	cond := ConditionNone
	vers := 0
	if ifopt := options["if"]; ifopt != "" {
		var err error
		cond, vers, err = ParseCondition(ifopt)
		if err != nil {
			return nil, err
		}
		delete(options, "if")
	}
	common := make([]*CommonAsset, len(names))
	for ii, v := range names {
		common[ii] = &CommonAsset{
			Manager:          m,
			name:             v,
			condition:        cond,
			conditionVersion: vers,
		}
	}
	return common, nil
}
