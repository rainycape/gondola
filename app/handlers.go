package app

// TemplateHandler returns a handler which executes the given
// template with the given data.
func TemplateHandler(name string, data interface{}) Handler {
	return func(ctx *Context) {
		ctx.MustExecute(name, data)
	}
}

// RedirectHandler returns a handler which redirects to the given
// url. The permanent argument indicates if the redirect should
// be temporary or permanent.
func RedirectHandler(destination string, permanent bool) Handler {
	return func(ctx *Context) {
		ctx.Redirect(destination, permanent)
	}
}

// StaticSitePattern is the pattern used in conjuction with StaticSiteHandler.
const StaticSitePattern = "^/(.*)$"

// StaticSiteHandler returns a handler which serves the template named
// by the request path (e.g. /foo will serve foo.html). The disabled
// argument can be used to always return a 404 for some paths, like
// templates which are only used as the base or included in another
// ones. The data argument is passed as is to template.Template.Execute.
// Usually, this handler should be used with StaticSitePattern.
func StaticSiteHandler(disabled []string, data interface{}) Handler {
	skip := make(map[string]struct{}, len(disabled))
	for _, v := range disabled {
		skip[v] = struct{}{}
	}
	exts := []string{".html", ".txt", ".md"}
	return func(ctx *Context) {
		name := ctx.IndexValue(0)
		if name == "" {
			name = "index"
		}
		if _, s := skip[name]; !s {
			app := ctx.App()
			for _, v := range exts {
				tmpl, err := app.LoadTemplate(name + v)
				if err == nil {
					if err := tmpl.Execute(ctx, data); err != nil {
						panic(err)
					}
					return
				}
			}
		}
		ctx.NotFound("page not found")
	}
}

// SignOutHandler can be added directly to an App. It signs out the
// current user (if any) and redirects back to the previous
// page unless the request was made via XHR.
func SignOutHandler(ctx *Context) {
	ctx.SignOut()
	if !ctx.IsXHR() {
		ctx.RedirectBack()
	}
}

// DataHandler is a handler than returns the data as an interface{}
// rather than sending it back to the client. A DataHandler can't be
// added directly to an App, it must be wrapped with a function which
// creates a Handler, like JSONHandler or ExecuteHandler.
type DataHandler func(*Context) (interface{}, error)

// JSONHandler returns a Handler which executes the given DataHandler
// to obtain the data and, if it succeeds, serializes the data using
// JSON and returns it back to the client.
func JSONHandler(dataHandler DataHandler) Handler {
	return func(ctx *Context) {
		data, err := dataHandler(ctx)
		if err != nil {
			panic(err)
		}
		if _, err := ctx.WriteJSON(data); err != nil {
			panic(err)
		}
	}
}

// ExecuteHandler returns a Handler which executes the given DataHandler
// to obtain the data and, if it succeeds, executes the given template
// passing it the obtained data.
func ExecuteHandler(dataHandler DataHandler, template string) Handler {
	return func(ctx *Context) {
		data, err := dataHandler(ctx)
		if err != nil {
			panic(err)
		}
		ctx.MustExecute(template, data)
	}
}
