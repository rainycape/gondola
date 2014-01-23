package app

import (
	"encoding/json"
	"gnd.la/app/debug"
	"time"
)

type debugInfo struct {
	Elapsed   time.Duration
	Timings   []*debug.Timing
	Remaining time.Duration
}

func debugInfoHandler(ctx *Context) {
	var info *debugInfo
	data := ctx.RequireFormValue("data")
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		panic(err)
	}
	info.Remaining = info.Elapsed
	for _, v := range info.Timings {
		info.Remaining -= v.Total()
	}
	t := newInternalTemplate(ctx.app)
	if err := t.Parse("debug_info.html"); err != nil {
		panic(err)
	}
	if err := t.tmpl.Compile(); err != nil {
		panic(err)
	}
	t.tmpl.MustExecute(ctx, info)
}
