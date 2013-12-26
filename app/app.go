// Package app provides a mux implementation which does
// regexp based URL routing and provides functions for
// managing the lifecycle of a request at different
// points.
package app

import (
	"bytes"
	"fmt"
	"gnd.la/app/cookies"
	"gnd.la/blobstore"
	"gnd.la/cache"
	"gnd.la/defaults"
	"gnd.la/loaders"
	"gnd.la/log"
	"gnd.la/orm"
	"gnd.la/signal"
	"gnd.la/template"
	"gnd.la/template/assets"
	"gnd.la/util"
	"gnd.la/util/internal/runtimeutil"
	"net/http"
	"net/http/httputil"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// IPXHeaders are the default headers which are used to
	// read the client's IP, in decreasing priority order.
	// You might change them if e.g. your CDN provider uses
	// different ones. Note that, for these values to have
	// any effect, the App needs to have TrustsXHeaders set
	// to true.
	IPXHeaders = []string{"X-Real-IP", "X-Forwarded-For"}
	// SchemeXHeaders are the scheme equivalent of IPXHeaders.
	SchemeXHeaders = []string{"X-Scheme", "X-Forwarded-Proto"}
)

type RecoverHandler func(*Context, interface{}) interface{}

// ContextProcessor functions run before the request is matched to
// a Handler and might alter the context in any way they see fit
type ContextProcessor func(*Context) bool

// ErrorHandler is called before an error is sent
// to the client. The parameters are the current context,
// the error message and the error code. If the handler
// returns true, the error is considered as handled and
// no further data is sent to the client.
type ErrorHandler func(*Context, string, int) bool

// LanguageHandler is use to determine the language
// when serving a request. See function SetLanguageHandler()
// in App.
type LanguageHandler func(*Context) string

type handlerInfo struct {
	host      string
	name      string
	path      string
	pathMatch []int
	re        *regexp.Regexp
	handler   Handler
}

const (
	poolSize = 16
)

var (
	devStatusPage  = "/_gondola_dev_server_status"
	monitorPage    = "/_gondola_monitor"
	monitorAPIPage = "/_gondola_monitor_api"
	assetsPrefix   = "/_gondola_assets"
)

type App struct {
	ContextProcessors    []ContextProcessor
	ContextFinalizers    []ContextFinalizer
	RecoverHandlers      []RecoverHandler
	handlers             []*handlerInfo
	trustXHeaders        bool
	appendSlash          bool
	errorHandler         ErrorHandler
	languageHandler      LanguageHandler
	secret               string
	encryptionKey        string
	defaultLanguage      string
	defaultCookieOptions *cookies.Options
	userFunc             UserFunc
	assetsManager        assets.Manager
	templatesLoader      loaders.Loader
	templatesMutex       sync.RWMutex
	templatesCache       map[string]Template
	templateProcessors   []TemplateProcessor
	templateVars         map[string]interface{}
	templateVarFuncs     map[string]reflect.Value
	debug                bool
	started              time.Time
	address              string
	port                 int
	mu                   sync.Mutex
	c                    *Cache
	o                    *Orm
	store                *blobstore.Store

	// Logger to use when logging requests. By default, it's
	// gnd.la/log.Std, but you can set it to nil to avoid
	// logging at all and gain a bit more of performance.
	Logger      *log.Logger
	contextPool chan *Context
}

// Handle is a shorthand for HandleOptions, passing nil as the Options.
func (app *App) Handle(pattern string, handler Handler) {
	app.HandleOptions(pattern, handler, nil)
}

// HandleOptions adds a new handler to the App. If the Options include a
// non-empty name, it can be be reversed using Context.Reverse or
// the "reverse" template function. To add a host-specific Handler,
// set the Host field in Options to a non-empty string.
func (app *App) HandleOptions(pattern string, handler Handler, opts *Options) {
	if handler == nil {
		panic(fmt.Errorf("handler for pattern %q can't be nil", pattern))
	}
	re := regexp.MustCompile(pattern)
	var host string
	var name string
	if opts != nil {
		host = opts.Host
		name = opts.Name
	}
	info := &handlerInfo{
		host:    host,
		name:    name,
		re:      re,
		handler: handler,
	}
	if p := literalRegexp(re); p != "" {
		info.path = p
		info.pathMatch = []int{0, len(p)}
	}
	app.handlers = append(app.handlers, info)
}

// AddContextProcessor adds context processor to the App.
// Context processors run in the same order they were added
// before the app starts matching the request to a handler and
// may alter the request in any way they see fit as well as
// writing to the context. If any of the processors returns
// true, the request is considered as served and no further
// processing to it is done.
func (app *App) AddContextProcessor(cp ContextProcessor) {
	app.ContextProcessors = append(app.ContextProcessors, cp)
}

// AddContextFinalizer adds a context finalizer to the app.
// Context finalizers run in the order they were added and
// after the request has been served (even if it was stopped by
// a context processor). They're intended as a way to perform
// some logging or cleanup (e.g. closing database connections
// that might have been opened during the request lifetime)
func (app *App) AddContextFinalizer(cf ContextFinalizer) {
	app.ContextFinalizers = append(app.ContextFinalizers, cf)
}

// AddRecoverHandler adds a recover handler to the app.
// Recover handlers are executed in the order they were added in case
// there's a panic while serving a request. The handlers may write
// to the context. If any recover handler returns nil the error is
// considered as handled and no panic is raised.
func (app *App) AddRecoverHandler(rh RecoverHandler) {
	app.RecoverHandlers = append(app.RecoverHandlers, rh)
}

// TrustsXHeaders returns if the app uses X headers
// for determining the remote IP and scheme. See SetTrustXHeaders()
// for a more detailed explanation.
func (app *App) TrustsXHeaders() bool {
	return app.trustXHeaders
}

// SetTrustXHeaders sets if the app uses X headers like
// X-Real-IP, X-Forwarded-For, X-Scheme and X-Forwarded-Proto
// to override the remote IP and scheme. This is useful
// when running your application behind a proxy or load balancer.
// The default is disabled. Please, keep in mind that enabling
// XHeaders processing when not running behind a proxy or load
// balancer which sanitizes the input *IS A SECURITY RISK*.
func (app *App) SetTrustXHeaders(t bool) {
	app.trustXHeaders = t
}

// AppendSlash returns if the app will automatically append
// a slash when appropriate. See SetAppendSlash for a more
// detailed description.
func (app *App) AppendsSlash() bool {
	return app.appendSlash
}

// SetAppendSlash enables or disables automatic slash appending.
// When enabled, GET and HEAD requests for /foo will be
// redirected to /foo/ if there's a valid handler for that URL,
// rather than returning a 404. The default is true.
func (app *App) SetAppendSlash(b bool) {
	app.appendSlash = b
}

// Secret returns the secret for this app. See
// SetSecret() for further details.
func (app *App) Secret() string {
	return app.secret
}

// SetSecret sets the secret associated with this app,
// which is used for signed cookies. It should be a
// random string with at least 32 characters. When the
// app is initialized, this value is set to the value
// returned by defaults.Secret() (which can be controlled
// from the config).
func (app *App) SetSecret(secret string) {
	app.secret = secret
}

// EncryptionKey returns the encryption key for this
// app. See SetEncryptionKey() for details.
func (app *App) EncryptionKey() string {
	return app.encryptionKey
}

// SetEncriptionKey sets the encryption key for this
// app, which is used by encrypted cookies. It should
// be a random string of 16, 24 or 32 characters.
func (app *App) SetEncryptionKey(key string) {
	app.encryptionKey = key
}

// DefaultCookieOptions returns the default options
// used for cookies. This is initialized to the value
// returned by cookies.Defaults(). See gnd.la/app/cookies
// documentation for more details.
func (app *App) DefaultCookieOptions() *cookies.Options {
	return app.defaultCookieOptions
}

// SetDefaultCookieOptions sets the default cookie options
// for this app. See gnd.la/cookies documentation for more
// details.
func (app *App) SetDefaultCookieOptions(o *cookies.Options) {
	app.defaultCookieOptions = o
}

// ErrorHandler returns the error handler (if any)
// associated with this app
func (app *App) ErrorHandler() ErrorHandler {
	return app.errorHandler
}

// SetErrorHandler sets the error handler for this app.
// See the documentation on ErrorHandler for a more
// detailed description.
func (app *App) SetErrorHandler(handler ErrorHandler) {
	app.errorHandler = handler
}

// LanguageHandler returns the language handler for this
// app. See SetLanguageHandler() for further information
// about language handlers.
func (app *App) LanguageHandler() LanguageHandler {
	return app.languageHandler
}

// SetLanguageHandler sets the language handler for this app.
// The LanguageHandler is responsible for determining the language
// used in translations for a request. If the empty string is returned
// the strings won't be translated. Finally, when a app does not have
// a language handler it uses the language specified by gnd.la/defaults.
func (app *App) SetLanguageHandler(handler LanguageHandler) {
	app.languageHandler = handler
}

func (app *App) UserFunc() UserFunc {
	return app.userFunc
}

func (app *App) SetUserFunc(f UserFunc) {
	app.userFunc = f
}

// AssetsManager returns the manager for static assets
func (app *App) AssetsManager() assets.Manager {
	return app.assetsManager
}

// SetAssetsManager sets the static assets manager for the app. See
// the documention on gnd.la/assets/Manager for further information.
func (app *App) SetAssetsManager(manager assets.Manager) {
	manager.SetDebug(app.Debug())
	app.assetsManager = manager
}

// TemplatesLoader returns the loader for the templates assocciated
// with this app. By default, templates will be loaded from the
// tmpl directory relative to the application binary.
func (app *App) TemplatesLoader() loaders.Loader {
	return app.templatesLoader
}

// SetTemplatesLoader sets the loader used to load the templates
// associated with this app. By default, templates will be loaded from the
// tmpl directory relative to the application binary.
func (app *App) SetTemplatesLoader(loader loaders.Loader) {
	app.templatesLoader = loader
}

// AddTemplateProcessor adds a new template processor. Template processors
// may modify a template after it's been loaded.
func (app *App) AddTemplateProcessor(processor TemplateProcessor) {
	app.templateProcessors = append(app.templateProcessors, processor)
}

// AddTemplateVars adds additional variables which will be passed
// to the templates executed by this app. The values in the map might
// either be values or functions which receive a *Context instance and return
// either one or two values (the second one must be an error), in which case
// they will be called with the current context to obtain the variable
// that will be passed to the template. You must call this
// function before any templates have been compiled. The value for
// each variable in the map is its default value, and it can
// be overriden by using ExecuteVars() rather than Execute() when
// executing the template.
func (app *App) AddTemplateVars(vars template.VarMap) {
	if app.templateVars == nil {
		app.templateVars = make(template.VarMap)
		app.templateVarFuncs = make(map[string]reflect.Value)
	}
	for k, v := range vars {
		if app.isReservedVariable(k) {
			panic(fmt.Errorf("Variable %s is reserved", k))
		}
		if t := reflect.TypeOf(v); t.Kind() == reflect.Func {
			inType := reflect.TypeOf(&Context{})
			if t.NumIn() != 1 || t.In(0) != inType {
				panic(fmt.Errorf("Template variable functions must receive a single %s argument", inType))
			}
			if t.NumOut() > 2 {
				panic(fmt.Errorf("Template variable functions must return at most 2 arguments"))
			}
			if t.NumOut() == 2 {
				o := t.Out(1)
				if o.Kind() != reflect.Interface || o.Name() != "error" {
					panic(fmt.Errorf("Template variable functions must return an error as their second argument"))
				}
			}
			app.templateVarFuncs[k] = reflect.ValueOf(v)
		} else {
			app.templateVars[k] = v
		}
	}
}

// LoadTemplate loads a template using the template
// loader and the asset manager assocciated with
// this app
func (app *App) LoadTemplate(name string) (Template, error) {
	app.templatesMutex.RLock()
	tmpl := app.templatesCache[name]
	app.templatesMutex.RUnlock()
	if tmpl == nil {
		t := newAppTemplate(app)
		vars := make(template.VarMap, len(app.templateVars)+len(app.templateVarFuncs))
		for k, v := range app.templateVars {
			vars[k] = v
		}
		for k, _ := range app.templateVarFuncs {
			vars[k] = nil
		}
		err := t.ParseVars(name, vars)
		if err != nil {
			return nil, err
		}
		for _, v := range app.templateProcessors {
			t.Template, err = v(t.Template)
			if err != nil {
				return nil, err
			}
		}
		tmpl = t
		if !app.debug {
			app.templatesMutex.Lock()
			app.templatesCache[name] = tmpl
			app.templatesMutex.Unlock()
		}
	}
	return tmpl, nil
}

// Debug returns if the app is in debug mode
// (i.e. templates are not cached).
func (app *App) Debug() bool {
	return app.debug
}

// SetDebug sets the debug state for the app.
// When true, templates executed via Context.Execute or
// Context.MustExecute() are recompiled every time
// they are executed. The default is the value
// returned by defaults.Debug() when the app is
// constructed. See the documentation on gnd.la/defaults
// for further information.
func (app *App) SetDebug(debug bool) {
	app.debug = debug
}

// Address returns the address this app is configured to listen
// on. By default, it's empty, meaning the app will listen on
// all interfaces.
func (app *App) Address() string {
	return app.address
}

// SetAddress changes the address this app will listen on.
func (app *App) SetAddress(address string) {
	app.address = address
}

// Port returns the port this app is configured to listen on.
// By default, it's initialized with the value returned by Port()
// in the defaults package (which can be altered using Gondola's
// config).
func (app *App) Port() int {
	return app.port
}

// SetPort sets the port on which this app will listen on. It's not
// recommended to call this function manually. Instead, use gnd.la/config
// to change the default port before creating the app. Otherwise, Gondola's
// development server won't work correctly.
func (app *App) SetPort(port int) {
	app.port = port
}

// HandleAssets adds several handlers to the app which handle
// assets efficiently and allows the use of the "assset"
// function from the templates. This function will also modify the
// asset loader associated with this app. prefix might be a relative
// (e.g. /static/) or absolute (e.g. http://static.example.com/) url
// while dir should be the path to the directory where the static
// assets reside. You probably want to use RelativePath() in gnd.la/util
// to define the directory relative to the application binary. Note
// that /favicon.ico and /robots.txt will be handled too, but they
// will must be in the directory which contains the rest of the assets.
func (app *App) HandleAssets(prefix string, dir string) {
	loader := loaders.FSLoader(dir)
	manager := assets.NewManager(loader, prefix)
	app.SetAssetsManager(manager)
	app.addAssetsManager(manager, true)
}

func (app *App) addAssetsManager(manager assets.Manager, main bool) {
	assetsHandler := assets.Handler(manager)
	handler := func(ctx *Context) {
		assetsHandler(ctx, ctx.R)
	}
	app.Handle("^"+manager.Prefix(), handler)
	if main {
		app.Handle("^/favicon.ico$", handler)
		app.Handle("^/robots.txt$", handler)
	}
}

// MustReverse calls Reverse and panics if it finds an error. See
// Reverse for further details.
func (app *App) MustReverse(name string, args ...interface{}) string {
	rev, err := app.Reverse(name, args...)
	if err != nil {
		panic(err)
	}
	return rev
}

// Reverse obtains the url given a handler name and its arguments.
// The number of arguments must be equal to the
// number of captured parameters in the patttern for the handler
// e.g. given the pattern ^/article/\d+/[\w\-]+/$, you should provide
// 2 arguments and passing 42 and "the-ultimate-answer-to-life-the-universe-and-everything"
// would return "/article/42/the-ultimate-answer-to-life-the-universe-and-everything/"
// If the handler is also restricted to a given hostname, the return value
// will be a scheme relative url e.g. //www.example.com/article/...
func (app *App) Reverse(name string, args ...interface{}) (string, error) {
	if name == "" {
		return "", fmt.Errorf("No handler name specified")
	}
	for _, v := range app.handlers {
		if v.name == name {
			reversed, err := formatRegexp(v.re, true, args...)
			if err != nil {
				if acerr, ok := err.(*argumentCountError); ok {
					if acerr.MinArguments == acerr.MaxArguments {
						return "", fmt.Errorf("Handler %q requires exactly %d arguments, %d received instead",
							name, acerr.MinArguments, len(args))
					}
					return "", fmt.Errorf("Handler %q requires at least %d arguments and at most %d arguments, %d received instead",
						name, acerr.MinArguments, acerr.MaxArguments, len(args))
				}
				return "", fmt.Errorf("Error reversing handler %q: %s", name, err)
			}
			if v.host != "" {
				reversed = fmt.Sprintf("//%s%s", v.host, reversed)
			}
			return reversed, nil
		}
	}
	return "", fmt.Errorf("No handler named %q", name)
}

// ListenAndServe starts listening on the configured address and
// port (see Address() and Port()).
// This function is a shortcut for
// http.ListenAndServe(app.Address()+":"+strconv.Itoa(app.Port()), app)
func (app *App) ListenAndServe() error {
	signal.Emit(signal.APP_WILL_LISTEN, app)
	app.started = time.Now().UTC()
	if app.Logger != nil && os.Getenv("GONDOLA_DEV_SERVER") == "" {
		if app.address != "" {
			app.Logger.Infof("Listening on %s, port %d", app.address, app.port)
		} else {
			app.Logger.Infof("Listening on port %d", app.port)
		}
	}
	return http.ListenAndServe(app.address+":"+strconv.Itoa(app.port), app)
}

// MustListenAndServe works like ListenAndServe, but panics if
// there's an error
func (app *App) MustListenAndServe() {
	err := app.ListenAndServe()
	if err != nil {
		log.Panicf("error listening on port %d: %s", app.port, err)
	}
}

// Cache returns this app's cache connection, using
// cache.NewDefault(). Use gnd.la/config or gnd.la/defaults
// to change the default cache. When the app
// is in debug mode, a new cache instance is returned
// every time. Otherwise, the cache instance is shared
// among all goroutines. Cache access is thread safe, but
// some methods (like NumQueries()) will be completely
// inaccurate because they will count all the queries made
// since the app initialization.
func (app *App) Cache() (*Cache, error) {
	if app.c == nil {
		app.mu.Lock()
		defer app.mu.Unlock()
		if app.c == nil {
			c, err := cache.New(defaults.Cache())
			if err != nil {
				return nil, err
			}
			app.c = &Cache{Cache: c, debug: app.debug}
			if app.debug {
				c := app.c
				app.c = nil
				return c, nil
			}
		}
	}
	return app.c, nil
}

// App returns this app's ORM connection, using the
// default database parameters. Use gnd.la/config or gnd.la/defaults
// to change the default ORM. When the app is in debug mode, a new
// ORM instance is returned every time. Otherwise, the app instance
// is shared amoung all goroutines. ORM usage is thread safe, but
// some methods (like NumQueries()) will be completely inaccurate
// because they wull count all the queries made since the app
// initialization.
func (app *App) Orm() (*Orm, error) {
	if app.o == nil {
		app.mu.Lock()
		defer app.mu.Unlock()
		if app.o == nil {
			url := defaults.Database()
			if url == nil {
				return nil, fmt.Errorf("default database is not set")
			}
			o, err := orm.New(url)
			if err != nil {
				return nil, err
			}
			app.o = &Orm{Orm: o, debug: app.debug}
			if app.debug {
				o := app.o
				o.SetLogger(log.Std)
				app.o = nil
				return o, nil
			}
		}
	}
	return app.o, nil
}

// Blobstore returns a blobstore using the default blobstore
// parameters. Use gnd.la/config or gnd.la/defaults to change
// the default blobstore. See gnd.la/blobstore for further
// information on using the blobstore.
func (app *App) Blobstore() (*blobstore.Store, error) {
	if app.store == nil {
		app.mu.Lock()
		defer app.mu.Unlock()
		if app.store == nil {
			var err error
			app.store, err = blobstore.New(defaults.Blobstore())
			if err != nil {
				return nil, err
			}
		}
	}
	return app.store, nil
}

func (app *App) readXHeaders(r *http.Request) {
	for _, v := range IPXHeaders {
		if value := r.Header.Get(v); value != "" {
			r.RemoteAddr = value
			break
		}
	}
	for _, v := range SchemeXHeaders {
		if value := r.Header.Get(v); value != "" {
			r.URL.Scheme = value
			// When setting the scheme, set also the host, otherwise
			// the url becomes invalid.
			r.URL.Host = r.Host
			break
		}
	}
}

func (app *App) handleHTTPError(ctx *Context, error string, code int) {
	defer app.recover(ctx)
	if app.errorHandler == nil || !app.errorHandler(ctx, error, code) {
		http.Error(ctx, error, code)
	}
}

func (app *App) handleError(ctx *Context, err interface{}) bool {
	if gerr, ok := err.(Error); ok {
		log.Debugf("HTTP error: %s (%d)", gerr.Error(), gerr.StatusCode())
		app.handleHTTPError(ctx, gerr.Error(), gerr.StatusCode())
		return true
	}
	return false
}

func (app *App) recover(ctx *Context) {
	if err := recover(); err != nil {
		app.recoverErr(ctx, err)
	}
}

func (app *App) recoverErr(ctx *Context, err interface{}) {
	if isIgnorable(err) {
		return
	}
	for _, v := range app.RecoverHandlers {
		err = v(ctx, err)
		if err == nil {
			break
		}
	}
	if err != nil && !app.handleError(ctx, err) {
		app.logError(ctx, err)
	}
}

func (app *App) logError(ctx *Context, err interface{}) {
	skip, stackSkip, _, _ := runtimeutil.GetPanic()
	var buf bytes.Buffer
	if ctx.R != nil {
		buf.WriteString("Panic serving ")
		buf.WriteString(ctx.R.Method)
		buf.WriteByte(' ')
		buf.WriteString(ctx.R.Host)
		buf.WriteString(ctx.R.URL.Path)
		if rq := ctx.R.URL.RawQuery; rq != "" {
			buf.WriteByte('?')
			buf.WriteString(rq)
		}
		if rf := ctx.R.URL.Fragment; rf != "" {
			buf.WriteByte('#')
			buf.WriteString(rf)
		}
		buf.WriteByte(' ')
		buf.WriteString(ctx.RemoteAddress())
		buf.WriteString(": ")
	} else {
		buf.WriteString("Panic: ")
	}
	buf.WriteString(fmt.Sprintf("%v", err))
	buf.WriteByte('\n')
	stack := runtimeutil.FormatStack(stackSkip)
	location, code := runtimeutil.FormatCaller(skip, 5, true, true)
	if location != "" {
		buf.WriteString("\n At ")
		buf.WriteString(location)
		if code != "" {
			buf.WriteByte('\n')
			buf.WriteString(code)
			buf.WriteByte('\n')
		}
	}
	if stack != "" {
		buf.WriteString("\nStack:\n")
		buf.WriteString(stack)
	}
	req := ""
	if ctx.R != nil {
		dump, derr := httputil.DumpRequest(ctx.R, true)
		if derr == nil {
			// This cleans up empty lines and replaces \r\n with \n
			req = util.Lines(string(dump), 0, 10000, true)
			buf.WriteString("\nRequest:\n")
			buf.WriteString(req)
		}
	}
	log.Error(buf.String())
	if app.debug {
		app.errorPage(ctx, skip, stackSkip, req, err)
	} else {
		app.handleHTTPError(ctx, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (app *App) errorPage(ctx *Context, skip int, stackSkip int, req string, err interface{}) {
	t := newInternalTemplate(app)
	if terr := t.Parse("panic.html"); terr != nil {
		panic(terr)
	}
	stack := runtimeutil.FormatStackHTML(stackSkip + 1)
	location, code := runtimeutil.FormatCallerHTML(skip+1, 5, true, true)
	ctx.statusCode = -http.StatusInternalServerError
	data := map[string]interface{}{
		"Error":    fmt.Sprintf("%v", err),
		"Location": location,
		"Code":     code,
		"Stack":    stack,
		"Request":  req,
		"Started":  strconv.FormatInt(app.started.Unix(), 10),
	}
	t.MustExecute(ctx, data)
}

// ServeHTTP is called from the net/http system. You shouldn't need
// to call this function
func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := app.newContext()
	ctx.ResponseWriter = w
	ctx.R = r
	defer app.closeContext(ctx)
	defer app.recover(ctx)
	if app.trustXHeaders {
		app.readXHeaders(r)
	}
	for _, v := range app.ContextProcessors {
		if v(ctx) {
			return
		}
	}

	if h := app.matchHandler(r, ctx); h != nil {
		h.handler(ctx)
		return
	}

	if app.appendSlash && (r.Method == "GET" || r.Method == "HEAD") && !strings.HasSuffix(r.URL.Path, "/") {
		r.URL.Path += "/"
		match := app.matchHandler(r, ctx)
		if match != nil {
			ctx.Redirect(r.URL.String(), true)
			r.URL.Path = r.URL.Path[:len(r.URL.Path)-1]
			return
		}
		r.URL.Path = r.URL.Path[:len(r.URL.Path)-1]
	}

	/* Not found */
	app.handleHTTPError(ctx, "Not Found", http.StatusNotFound)
}

func (app *App) matchHandler(r *http.Request, ctx *Context) *handlerInfo {
	p := r.URL.Path
	for _, v := range app.handlers {
		if v.host != "" && v.host != r.Host {
			continue
		}
		if v.path != "" {
			if v.path == p {
				ctx.reProvider.reset(v.re, p, v.pathMatch)
				ctx.handlerName = v.name
				return v
			}
		} else {
			// Use FindStringSubmatchIndex, since this way we can
			// reuse the slices used to store context arguments
			if m := v.re.FindStringSubmatchIndex(p); m != nil {
				ctx.reProvider.reset(v.re, p, m)
				ctx.handlerName = v.name
				return v
			}
		}
	}
	return nil
}

// newContext returns a new context, using the
// context pool when possible.
func (app *App) newContext() *Context {
	var ctx *Context
	select {
	case ctx = <-app.contextPool:
		ctx.reset()
	default:
		p := &regexpProvider{}
		ctx = &Context{app: app, provider: p, reProvider: p, started: time.Now()}
	}
	return ctx
}

// NewContext initializes and returns a new context
// asssocciated with this app using the given ContextProvider
// to retrieve its arguments.
func (app *App) NewContext(p ContextProvider) *Context {
	return &Context{app: app, provider: p, started: time.Now()}
}

// CloseContext closes the passed context, which should have been
// created via NewContext(). Keep in mind that this function is
// called for you most of the time. As a rule of thumb, if you
// don't call NewContext() yourself, you don't need to call
// CloseContext().
func (app *App) CloseContext(ctx *Context) {
	for _, v := range app.ContextFinalizers {
		v(ctx)
	}
	ctx.Close()
	if app.Logger != nil && ctx.R != nil && ctx.R.URL.Path != devStatusPage && ctx.R.URL.Path != monitorAPIPage {
		// Log at most with Warning level, to avoid potentially generating
		// an email to the admin when running in production mode. If there
		// was an error while processing this request, it has been already
		// emailed to the admin, along the stack trace, in recover().
		level := log.LInfo
		if ctx.statusCode >= 400 {
			level = log.LWarning
		}
		app.Logger.Log(level, strings.Join([]string{ctx.R.Method, ctx.R.RequestURI, ctx.RemoteAddress(),
			strconv.Itoa(ctx.statusCode), ctx.Elapsed().String()}, " "))
	}

}

// closeContext calls CloseContexts and stores the context in
// in the pool for reusing it.
func (app *App) closeContext(ctx *Context) {
	app.CloseContext(ctx)
	select {
	case app.contextPool <- ctx:
	default:
	}
}

func (app *App) isReservedVariable(va string) bool {
	for _, v := range reservedVariables {
		if v == va {
			return true
		}
	}
	return false
}

// Returns a new App initialized with the current default values.
// See gnd.la/defaults for further information. Keep in mind that,
// for performance reasons, the values from gnd.la/defaults are
// copied to the app when it's created, so any changes made to
// gnd.la/defaults after app creation won't have any effect on it.
func New() *App {
	m := &App{
		debug:           defaults.Debug(),
		port:            defaults.Port(),
		secret:          defaults.Secret(),
		encryptionKey:   defaults.EncryptionKey(),
		defaultLanguage: defaults.Language(),
		appendSlash:     true,
		templatesCache:  make(map[string]Template),
		Logger:          log.Std,
		contextPool:     make(chan *Context, poolSize),
	}
	// Used to automatically reload the page on panics when the server
	// is restarted.
	if m.debug {
		m.Handle(devStatusPage, func(ctx *Context) {
			ctx.WriteJson(map[string]interface{}{
				"built":   nil,
				"started": strconv.FormatInt(m.started.Unix(), 10),
			})
		})
		m.Handle(monitorAPIPage, monitorAPIHandler)
		m.Handle(monitorPage, monitorHandler)
		m.addAssetsManager(internalAssetsManager(), false)
	}
	return m
}
