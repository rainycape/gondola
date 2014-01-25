package app

import (
	"bytes"
	"compress/flate"
	"encoding/json"
	"gnd.la/app/profile"
	"gnd.la/encoding/base64"
	"gnd.la/util/cryptoutil"
	"strconv"
	"strings"
	"time"
)

type profileInfo struct {
	Elapsed   time.Duration     `json:"e"`
	Timings   []*profile.Timing `json:"t"`
	Remaining time.Duration     `json:"-"`
}

func getProfileInfo(ctx *Context) *profileInfo {
	return &profileInfo{Elapsed: ctx.Elapsed(), Timings: profile.Timings()}
}

func profileHeader(ctx *Context) string {
	data, _ := json.Marshal(getProfileInfo(ctx))
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	w.Write(data)
	w.Close()
	return base64.Encode(buf.Bytes())
}

func shouldProfile(ctx *Context) bool {
	if req := ctx.R.Header.Get(profile.HeaderName); req != "" {
		if inDevServer {
			return true
		}
		if req == "true" {
			ctx.Header().Add(profile.HeaderName, "auth")
			return false
		}
		signer := cryptoutil.Signer{Salt: []byte(profile.Salt), Key: []byte(ctx.app.Secret)}
		data, err := signer.Unsign(req)
		if err == nil {
			parts := strings.Split(string(data), ":")
			if len(parts) == 2 {
				key := "profile-" + parts[1]
				var seen bool
				if ctx.Cache().Get(key, &seen) == nil && seen {
					return false
				}
				ts, err := strconv.ParseInt(parts[0], 10, 64)
				if err == nil {
					delta := time.Now().Unix() - ts
					if delta >= -300 && delta <= 300 {
						ctx.Cache().Set(key, true, 300)
						return true
					}
				}
			}
		}
		ctx.Header().Add(profile.HeaderName, "denied")
	}
	return false
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
