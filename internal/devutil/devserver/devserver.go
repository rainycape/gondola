package devserver

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"

	"gnd.la/internal/devutil/devassets"
	"gnd.la/kvs"
	"gnd.la/template"
	"gnd.la/template/assets"
)

const (
	EnvVar = "GONDOLA_INSIDE_DEV_SERVER"
)

// IsActive returns true iff the app is running from
// the gondola development server.
func IsActive() bool {
	return os.Getenv(EnvVar) != ""
}

type devServerKeyType int

const (
	devServerKey devServerKeyType = iota
)

func IsDevServer(s kvs.Storage) bool {
	val, _ := s.Get(devServerKey).(bool)
	return val
}

func SetIsDevServer(s kvs.Storage, value bool) {
	s.Set(devServerKey, value)
}

func ReloadHook() *template.Hook {
	reload := template.New(devassets.AssetsFS, nil)
	if err := reload.Parse("reload.html"); err != nil {
		panic(err)
	}
	return &template.Hook{
		Template: reload,
		Position: assets.Bottom,
	}
}

func UpdatesPath() string {
	return "/_gondola_dev_server_updates"
}

// Used to avoid circular imports
type appContext interface {
	URL() *url.URL
	Request() *http.Request
}

func updatesURL(val reflect.Value) (url string, reason string) {
	if ctx, ok := val.Interface().(appContext); ok && ctx != nil {
		if u := ctx.URL(); u != nil {
			if req := ctx.Request(); req != nil {
				if req.Method != "GET" {
					return "", fmt.Sprintf("request method was %v, not GET", req.Method)
				}
				cpy := *u
				switch u.Scheme {
				case "http":
					cpy.Scheme = "ws"
				case "https":
					cpy.Scheme = "wss"
				}
				cpy.Path = UpdatesPath()
				cpy.RawQuery = ""
				return cpy.String(), ""
			}
		}
	}
	return "", ""
}

func TemplateVars(typ interface{}) template.VarMap {
	fin := []reflect.Type{reflect.TypeOf(typ)}
	stringOut := []reflect.Type{reflect.TypeOf("")}
	stringFuncTyp := reflect.FuncOf(fin, stringOut, false)
	varFunc := func(in []reflect.Value) []reflect.Value {
		url, _ := updatesURL(in[0])
		return []reflect.Value{reflect.ValueOf(url)}
	}
	boolOut := []reflect.Type{reflect.TypeOf(true)}
	boolFuncTyp := reflect.FuncOf(fin, boolOut, false)
	enabledFunc := func(in []reflect.Value) []reflect.Value {
		url, _ := updatesURL(in[0])
		enabled := url != ""
		return []reflect.Value{reflect.ValueOf(enabled)}
	}
	template.AddFunc(&template.Func{
		Name:   "__gondola_is_live_reload_enabled",
		Fn:     reflect.MakeFunc(boolFuncTyp, enabledFunc).Interface(),
		Traits: template.FuncTraitContext,
	})
	reasonFunc := func(in []reflect.Value) []reflect.Value {
		_, reason := updatesURL(in[0])
		return []reflect.Value{reflect.ValueOf(reason)}
	}
	template.AddFunc(&template.Func{
		Name:   "__gondola_is_live_reload_disabled_reason",
		Fn:     reflect.MakeFunc(stringFuncTyp, reasonFunc).Interface(),
		Traits: template.FuncTraitContext,
	})

	return template.VarMap{
		"BroadcasterWebsocketUrl": reflect.MakeFunc(stringFuncTyp, varFunc).Interface(),
	}
}
