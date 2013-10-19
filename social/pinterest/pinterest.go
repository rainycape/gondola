package pinterest

import (
	"gnd.la/util/textutil"
)

type Credentials struct {
	Username string
	Password string
}

func (c *Credentials) Parse(raw string) error {
	fields, err := textutil.SplitFieldsOptions(raw, ":", &textutil.SplitOptions{Count: 2})
	if err != nil {
		return err
	}
	c.Username = fields[0]
	c.Password = fields[1]
	return nil
}

func Pin(url string, image_url string, board string, cred *Credentials) error {
	return nil
}
