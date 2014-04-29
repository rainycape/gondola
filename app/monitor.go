package app

import (
	"runtime"
)

func monitorHandler(ctx *Context) {
	t := newInternalTemplate(ctx.app)
	if err := t.parse("monitor.html", nil); err != nil {
		panic(err)
	}
	if err := t.prepare(); err != nil {
		panic(err)
	}
	if err := t.Execute(ctx, nil); err != nil {
		panic(err)
	}
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
