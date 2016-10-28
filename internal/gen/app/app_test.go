package app

import (
	"testing"
)

func TestApp(t *testing.T) {
	app, err := Parse("/home/fiam/go/src/gnd.la/apps/articles")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Gen(false); err != nil {
		t.Error(err)
	}
}
