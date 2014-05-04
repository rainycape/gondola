package facebook

import (
	"fmt"

	"gnd.la/net/httpclient"
	"gnd.la/util/stringutil"
)

type App struct {
	Id         string
	Secret     string
	Client     *httpclient.Client
	httpClient *httpclient.Client
}

func (a *App) Parse(s string) error {
	fields, err := stringutil.SplitFields(s, ":")
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

func (a *App) Clone(ctx httpclient.Context) *App {
	ac := *a
	ac.Client = ac.Client.Clone(ctx)
	return &ac
}

func (a *App) client() *httpclient.Client {
	if a.Client != nil {
		return a.Client
	}
	if a.httpClient == nil {
		a.httpClient = httpclient.New(nil)
	}
	return a.httpClient
}
