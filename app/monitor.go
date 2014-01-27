package app

import (
	"runtime"
)

func monitorHandler(ctx *Context) {
	t := newInternalTemplate(ctx.app)
	if err := t.Parse("monitor.html"); err != nil {
		panic(err)
	}
	if err := t.tmpl.Compile(); err != nil {
		panic(err)
	}
	t.tmpl.MustExecute(ctx, nil)
}

func monitorAPIHandler(ctx *Context) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	data := map[string]interface{}{
		"mem": &stats,
	}
	if _, err := ctx.WriteJSON(data); err != nil {
		panic(err)
	}
}
