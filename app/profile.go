package app

import (
	"encoding/json"
	"gnd.la/app/profile"
	"time"
)

type profileInfo struct {
	Elapsed   time.Duration
	Timings   []*profile.Timing
	Remaining time.Duration
}

func profileInfoHandler(ctx *Context) {
	var info *profileInfo
	data := ctx.RequireFormValue("data")
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		panic(err)
	}
	info.Remaining = info.Elapsed
	for _, v := range info.Timings {
		info.Remaining -= v.Total()
	}
	t := newInternalTemplate(ctx.app)
	if err := t.Parse("profile_info.html"); err != nil {
		panic(err)
	}
	if err := t.tmpl.Compile(); err != nil {
		panic(err)
	}
	t.tmpl.MustExecute(ctx, info)
}
