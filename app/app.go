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
	"gnd.la/util/hashutil"
	"gnd.la/util/internal/runtimeutil"
	"gnd.la/util/internal/templateutil"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template/parse"
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
// when serving a request. See App.SetLanguageHandler().
type LanguageHandler func(*Context) string

type handlerInfo struct {
	host      string
	name      string
	path      string
	pathMatch []int
	re        *regexp.Regexp
	handler   Handler
}

type includedApp struct {
	prefix  string
	app     *App
	base    string
	main    string
	renames map[string]string
}

func (a *includedApp) assetFuncName() string {
	return strings.ToLower(a.app.name) + "_" + template.AssetFuncName
}

func (a *includedApp) assetFunc(t *template.Template) func(string) (string, error) {
	return func(arg string) (string, error) {
		name := a.renames[arg]
		return t.Asset(name)
	}
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
	name                 string
	defaultCookieOptions *cookies.Options
	userFunc             UserFunc
	assetsManager        *assets.Manager
	templatesLoader      loaders.Loader
	templatesMutex       sync.RWMutex
	templatesCache       map[string]Template
	templateProcessors   []TemplateProcessor
	namespace            *namespace
	hooks                []*template.Hook
	debug                bool
	started              time.Time
	address              string
	port                 int
	mu                   sync.Mutex
	c                    *Cache
	o                    *Orm
	store                *blobstore.Store

	// Used for included apps
	included  []*includedApp
	parent    *App
	childInfo *includedApp

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

// HandleNamed is a shorthand for HandleOptions, passing an Options instance
// with just the name set.
func (app *App) HandleNamed(pattern string, handler Handler, name string) {
	app.HandleOptions(pattern, handler, &Options{Name: name})
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

func (app *App) Include(prefix string, included *App, base string, main string) {
	if err := app.include(prefix, included, base, main); err != nil {
		panic(err)
	}
}

func (app *App) include(prefix string, child *App, base string, main string) error {
	if child.parent != nil {
		return fmt.Errorf("app %v already has been included in another app", child)
	}
	if child.name == "" {
		return fmt.Errorf("included app %v can't have an empty name", child)
	}
	// prefix must start with / and end without /,
	// fix it if it doesn't match
	if prefix == "" || prefix[0] != '/' {
		prefix = "/" + prefix
	}
	for prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}
	for _, v := range app.included {
		if v.prefix == prefix {
			return fmt.Errorf("can't include app at prefix %q, app %q is already using it", prefix, v.app.name)
		}
		if v.app.name == child.name {
			return fmt.Errorf("duplicate app name %q", v.app.name)
		}
	}
	child.SetDebug(app.debug)
	child.parent = app
	child.secret = app.secret
	child.encryptionKey = app.encryptionKey
	child.languageHandler = app.languageHandler
	child.userFunc = app.userFunc
	child.Logger = app.Logger
	included := &includedApp{
		prefix: prefix,
		app:    child,
		base:   base,
		main:   main,
	}
	child.childInfo = included
	if child.assetsManager != nil {
		if err := app.importAssets(included); err != nil {
			return fmt.Errorf("error importing %q assets: %s", child.name, err)
		}
	}
	for _, v := range child.hooks {
		app.rewriteAssets(v.Template, included)
		root := v.Template.Trees[v.Template.Root()]
		app.prepareNamespace(root, child.name)
		app.hooks = append(app.hooks, v)
	}
	app.included = append(app.included, included)
	if app.namespace == nil {
		app.namespace = &namespace{}
	}
	if child.namespace != nil {
		if err := app.namespace.addNs(child.name, child.namespace); err != nil {
			return err
		}
	}
	return nil
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
	for _, v := range app.included {
		v.app.secret = secret
	}
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
	for _, v := range app.included {
		v.app.encryptionKey = key
	}
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

func (app *App) Name() string {
	return app.name
}

func (app *App) SetName(name string) {
	if app.parent != nil {
		panic(fmt.Errorf("can't rename app %q, it has been already included", app.name))
	}
	app.name = name
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
	for _, v := range app.included {
		v.app.languageHandler = handler
	}
}

func (app *App) UserFunc() UserFunc {
	return app.userFunc
}

func (app *App) SetUserFunc(f UserFunc) {
	app.userFunc = f
	for _, v := range app.included {
		v.app.userFunc = f
	}
}

// AssetsManager returns the manager for static assets
func (app *App) AssetsManager() *assets.Manager {
	return app.assetsManager
}

// SetAssetsManager sets the static assets manager for the app. See
// the documention on gnd.la/template/assets.Manager for further information.
func (app *App) SetAssetsManager(manager *assets.Manager) {
	app.assetsManager = manager
}

// TemplatesLoader returns the loader for the templates assocciated
// with this app. By default, templates will be loaded from the
// tmpl directory relative to the application binary.
func (app *App) TemplatesLoader() loaders.Loader {
	if app.templatesLoader == nil {
		app.templatesLoader = template.DefaultTemplateLoader()
	}
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
// each provided variable is its default value, and it can
// be overriden by using ExecuteVars() rather than Execute() when
// executing the template.
func (app *App) AddTemplateVars(vars template.VarMap) {
	if app.namespace == nil {
		app.namespace = &namespace{}
	}
	if err := app.namespace.add(vars); err != nil {
		panic(err)
	}
}

func (app *App) AddHook(hook *template.Hook) {
	app.hooks = append(app.hooks, hook)
}

// LoadTemplate loads a template using the template
// loader and the asset manager assocciated with
// this app
func (app *App) LoadTemplate(name string) (Template, error) {
	app.templatesMutex.RLock()
	tmpl := app.templatesCache[name]
	app.templatesMutex.RUnlock()
	if tmpl == nil {
		t, err := app.loadTemplate(name)
		if err != nil {
			return nil, err
		}
		funcs := make(template.FuncMap)
		for _, v := range app.included {
			funcs[v.assetFuncName()] = v.assetFunc(t.tmpl)
		}
		t.tmpl.Funcs(funcs)
		for _, v := range app.hooks {
			if err := t.tmpl.Hook(v); err != nil {
				return nil, fmt.Errorf("error hooking %q: %s", v.Template.Root(), err)
			}
		}
		if err := t.tmpl.PrepareAssets(); err != nil {
			return nil, err
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

func (app *App) loadTemplate(name string) (*tmpl, error) {
	t := newAppTemplate(app)
	var vars map[string]interface{}
	if app.namespace != nil {
		var err error
		if vars, err = app.namespace.eval(nil); err != nil {
			return nil, err
		}
	}
	err := t.ParseVars(name, vars)
	if err != nil {
		return nil, err
	}
	for _, v := range app.templateProcessors {
		t.tmpl, err = v(t.tmpl)
		if err != nil {
			return nil, err
		}
	}
	if app.parent != nil {
		return app.parent.chainTemplate(t, app.childInfo)
	}
	return t, nil
}

func (app *App) prepareNamespace(tree *parse.Tree, ns string) {
	// Mangle the Tree to set $Vars = $Vars.Name
	// just before the with .Data node
	for ii, node := range tree.Root.Nodes {
		if _, ok := node.(*parse.WithNode); ok {
			var nodes []parse.Node
			nodes = append(nodes, tree.Root.Nodes[:ii]...)
			nodes = append(nodes, &parse.ActionNode{
				NodeType: parse.NodeAction,
				Pipe: &parse.PipeNode{
					NodeType: parse.NodePipe,
					Decl:     []*parse.VariableNode{{NodeType: parse.NodeVariable, Ident: []string{"$Vars"}}},
					Cmds: []*parse.CommandNode{{NodeType: parse.NodeCommand,
						Args: []parse.Node{
							&parse.FieldNode{NodeType: parse.NodeField, Ident: []string{"Vars", ns}},
						},
					}},
				},
			})
			nodes = append(nodes, tree.Root.Nodes[ii:]...)
			tree.Root.Nodes = nodes
			break
		}
	}
}

func (app *App) rewriteAssets(t *template.Template, included *includedApp) error {
	for _, group := range t.Assets() {
		for _, a := range group.Assets {
			if a.IsRemote() || a.IsHTML() {
				continue
			}
			name := included.renames[a.Name]
			if name == "" {
				return fmt.Errorf("asset %q referenced from template %q does not exist", a.Name, t.Name)
			}
			a.Name = name
		}
		group.Manager = app.assetsManager
	}
	fname := included.assetFuncName()
	for _, v := range t.Trees {
		templateutil.WalkTree(v, func(n, p parse.Node) {
			if n.Type() == parse.NodeIdentifier {
				id := n.(*parse.IdentifierNode)
				if id.Ident == template.AssetFuncName {
					id.Ident = fname
				}
			}
		})
	}
	return t.Rebuild()
}

func (app *App) chainTemplate(t *tmpl, included *includedApp) (*tmpl, error) {
	base, err := app.loadTemplate(included.base)
	if err != nil {
		return nil, err
	}
	if err := app.rewriteAssets(t.tmpl, included); err != nil {
		return nil, err
	}
	base.tmpl.AddAssets(t.tmpl.Assets())
	for k, v := range t.tmpl.Trees {
		// This will happen with built-in added templates, like
		// TopAssets, BottomAssets...
		if _, ok := base.tmpl.Trees[k]; ok {
			continue
		}
		name := k
		if name == t.tmpl.Root() {
			name = included.main
			app.prepareNamespace(v, included.app.name)
		}
		if err := base.tmpl.AddParseTree(name, v); err != nil {
			return nil, err
		}
	}
	return base, nil
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

func (app *App) addAssetsManager(manager *assets.Manager, main bool) {
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
	return app.reverse(name, args)
}

func (app *App) reverse(name string, args []interface{}) (string, error) {
	if name == "" {
		return "", fmt.Errorf("no handler name specified")
	}
	found, s, err := app.reverseHandler(name, args)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("no handler named %q", name)
	}
	return s, nil
}

func (app *App) reverseHandler(name string, args []interface{}) (bool, string, error) {
	for _, v := range app.handlers {
		if v.name == name {
			reversed, err := formatRegexp(v.re, true, args)
			if err != nil {
				if acerr, ok := err.(*argumentCountError); ok {
					if acerr.MinArguments == acerr.MaxArguments {
						return true, "", fmt.Errorf("handler %q requires exactly %d arguments, %d received instead",
							name, acerr.MinArguments, len(args))
					}
					return true, "", fmt.Errorf("handler %q requires at least %d arguments and at most %d arguments, %d received instead",
						name, acerr.MinArguments, acerr.MaxArguments, len(args))
				}
				return true, "", fmt.Errorf("error reversing handler %q: %s", name, err)
			}
			if app.childInfo != nil {
				// Don't use path.Join, it will remove any trailing
				// slashes. Since the prefix has been sanitized in
				// Include, we can just prepend it.
				reversed = app.childInfo.prefix + reversed
			}
			if v.host != "" {
				reversed = fmt.Sprintf("//%s%s", v.host, reversed)
			}
			return true, reversed, nil
		}
	}
	for _, v := range app.included {
		if found, s, err := v.app.reverseHandler(name, args); found {
			return found, s, err
		}
	}
	return false, "", nil
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
			if app.parent != nil {
				var err error
				app.c, err = app.parent.Cache()
				if err != nil {
					return nil, err
				}
			} else {
				c, err := cache.New(defaults.Cache())
				if err != nil {
					return nil, err
				}
				app.c = &Cache{Cache: c, debug: app.debug}
			}
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
			if app.parent != nil {
				var err error
				app.o, err = app.parent.Orm()
				if err != nil {
					return nil, err
				}
			} else {
				o, err := app.openOrm()
				if err != nil {
					return nil, err
				}
				app.o = &Orm{Orm: o, debug: app.debug}
			}
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

func (app *App) openOrm() (*orm.Orm, error) {
	if app.parent != nil {
		return app.parent.openOrm()
	}
	url := defaults.Database()
	if url == nil {
		return nil, fmt.Errorf("default database is not set")
	}
	return orm.New(url)
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
			if app.parent != nil {
				app.store, err = app.parent.Blobstore()
			} else {
				app.store, err = blobstore.New(defaults.Blobstore())
			}
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
	t.tmpl.MustExecute(ctx, data)
}

// ServeHTTP is called from the net/http system. You shouldn't need
// to call this function
func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := app.newContext(w, r)
	defer app.closeContext(ctx)
	defer app.recover(ctx)
	if app.runProcessors(ctx) {
		return
	}
	if !app.serve(r.URL.Path, ctx) {
		// Not Found
		app.handleHTTPError(ctx, "Not Found", http.StatusNotFound)
	}
}

func (app *App) serve(path string, ctx *Context) bool {
	if handler := app.matchHandler(path, ctx); handler != nil {
		handler(ctx)
		return true
	}

	for _, v := range app.included {
		if strings.HasPrefix(path, v.prefix) {
			ctx.app = v.app
			defer func() {
				ctx.app = app
			}()
			if v.app.serve(path[len(v.prefix):], ctx) {
				return true
			}
		}
	}

	if app.appendSlash && (ctx.R.Method == "GET" || ctx.R.Method == "HEAD") && !strings.HasSuffix(path, "/") {
		if app.matchHandler(path+"/", ctx) != nil {
			prevPath := ctx.R.URL.Path
			ctx.R.URL.Path += "/"
			ctx.Redirect(ctx.R.URL.String(), true)
			ctx.R.URL.Path = prevPath
			return true
		}
	}
	return false
}

func (app *App) matchHandler(path string, ctx *Context) Handler {
	for _, v := range app.handlers {
		if v.host != "" && v.host != ctx.R.Host {
			continue
		}
		if v.path != "" {
			if v.path == path {
				ctx.reProvider.reset(v.re, path, v.pathMatch)
				ctx.handlerName = v.name
				return v.handler
			}
		} else {
			// Use FindStringSubmatchIndex, since this way we can
			// reuse the slices used to store context arguments
			if m := v.re.FindStringSubmatchIndex(path); m != nil {
				ctx.reProvider.reset(v.re, path, m)
				ctx.handlerName = v.name
				return v.handler
			}
		}
	}
	return nil
}

// newContext returns a new context, using the
// context pool when possible.
func (app *App) newContext(w http.ResponseWriter, r *http.Request) *Context {
	var ctx *Context
	select {
	case ctx = <-app.contextPool:
		ctx.reset()
	default:
		p := &regexpProvider{}
		ctx = &Context{app: app, provider: p, reProvider: p, started: time.Now()}
	}
	ctx.ResponseWriter = w
	ctx.R = r
	if app.trustXHeaders {
		app.readXHeaders(r)
	}
	return ctx
}

func (app *App) runProcessors(ctx *Context) bool {
	for _, v := range app.ContextProcessors {
		if v(ctx) {
			app.closeContext(ctx)
			return true
		}
	}
	return false
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

func (app *App) importAssets(included *includedApp) error {
	if am := included.app.assetsManager; am != nil {
		m := app.assetsManager
		prefix := strings.ToLower(included.app.name)
		res, err := am.Loader().List()
		if err != nil {
			return err
		}
		renames := make(map[string]string)
		for _, v := range res {
			src, _, err := am.Load(v)
			if err != nil {
				return err
			}
			defer src.Close()
			sum := hashutil.Fnv32a(src)
			nonExt := v[:len(v)-len(path.Ext(v))]
			dest := path.Join(prefix, nonExt+".gen."+sum+path.Ext(v))
			renames[v] = dest
			if f, _, _ := m.Load(dest); f != nil {
				f.Close()
				continue
			}
			f, err := m.Create(dest, true)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := src.Seek(0, os.SEEK_SET); err != nil {
				return err
			}
			if _, err := io.Copy(f, src); err != nil {
				return err
			}
		}
		included.renames = renames
	}
	return nil
}

// New returns a new App initialized with the current default values.
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
