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

// SignOutHandler can be added directly to an App. It signs out the
// current user (if any) and redirects back to the previous
// page unless the request was made via XHR.
func SignOutHandler(ctx *Context) {
	ctx.SignOut()
	if !ctx.IsXHR() {
		ctx.RedirectBack()
	}
}
