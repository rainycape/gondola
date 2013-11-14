package facebook

import (
	"fmt"
	"gnd.la/util/textutil"
)

type App struct {
	Id     string
	Secret string
}

func (a *App) Parse(s string) error {
	fields, err := textutil.SplitFields(s, ":")
	if err != nil {
		return err
	}
	switch len(fields) {
	case 1:
		a.Id = fields[0]
	case 2:
		a.Id = fields[0]
		a.Secret = fields[1]
	default:
		return fmt.Errorf("invalid number of fields: %d", len(fields))
	}
	return nil
}
