package apputil

import "gnd.la/app"

type ReusableApp struct {
	*app.App
	Prefix       string
	BaseTemplate string
}

func NewReusableApp(name string) *ReusableApp {
	a := app.New()
	a.SetName(name)
	return &ReusableApp{
		App: a,
	}
}
