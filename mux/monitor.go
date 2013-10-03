package mux

import (
	"runtime"
)

func monitorHandler(ctx *Context) {
	t := newTemplate(ctx.mux, muxAssets)
	if err := t.Parse("monitor.html"); err != nil {
		panic(err)
	}
	t.MustExecute(ctx, nil)
}

func monitorAPIHandler(ctx *Context) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	data := map[string]interface{}{
		"mem": &stats,
	}
	if _, err := ctx.WriteJson(data); err != nil {
		panic(err)
	}
}
