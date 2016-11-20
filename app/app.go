// Package app provides a mux implementation which does
// regexp based URL routing and provides functions for
// managing the lifecycle of a request at different
// points.
package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template/parse"
	"time"

	"golang.org/x/net/websocket"

	"gnd.la/app/cookies"
	"gnd.la/app/profile"
	"gnd.la/blobstore"
	"gnd.la/cache"
	"gnd.la/crypto/cryptoutil"
	"gnd.la/crypto/hashutil"
	"gnd.la/encoding/codec"
	"gnd.la/internal"
	"gnd.la/internal/devutil/devserver"
	"gnd.la/internal/runtimeutil"
	"gnd.la/internal/templateutil"
	"gnd.la/kvs"
	"gnd.la/log"
	"gnd.la/net/mail"
	"gnd.la/orm"
	"gnd.la/signal"
	"gnd.la/template"
	"gnd.la/template/assets"
	"gnd.la/util/stringutil"

	"path/filepath"

	"github.com/rainycape/vfs"
)

const (
	// WILL_LISTEN is emitted just before a *gnd.la/app.App will
	// start listening. The object is the App.
	WILL_LISTEN = "gnd.la/app.will-listen"
	// DID_LISTEN is emitted after a *gnd.la/app.App starts
	// listening. The object is the App.
	DID_LISTEN = "gnd.la/app.did-listen"
	// WILL_PREPARE is emitted at the beginning of App.Prepare.
	// The object is the App.
	WILL_PREPARE = "gnd.la/app.will-prepare"
	// DID_PREPARE is emitted when App.Prepare ends without errors.
	// The object is the App.
	DID_PREPARE = "gnd.la/app.did-prepare"
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

	inDevServer bool

	errNoSecret           = errors.New("app has no secret")
	errNoKey              = errors.New("app has no encryption key")
	errNoDefaultDatabase  = errors.New("default database is not set")
	errNoDefaultBlobstore = errors.New("default blobstore is not set")
	errNoAppOrm           = errors.New("App.Orm() does not work on App Engine - use Context.Orm() instead")
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
	rc        *regexpCache
	handler   Handler
}

type includedApp struct {
	prefix    string
	app       *App
	container string
	renames   map[string]string
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
	monitorPage    = "/_gondola_monitor"
	monitorAPIPage = "/_gondola_monitor_api"
	assetsPrefix   = "/_gondola_assets"
)

// App is the central piece of a Gondola application. It routes
// requests, invokes the requires handlers, manages connections to
// the cache and the database, caches templates and stores most of
// the configurable parameters of a Gondola application. Use New()
// to initialize an App, since there are some private fields which
// require initialization.
//
// Cookie configuration fields should be set
// from your code, since there are no configuration
// options for them. Note that the defaults will work
// fine in most cases, but applications with special
// security or performance requirements might need to
// alter them. As with other configuration parameters, these
// can only be changed before the app adding any included apps
// and must not be changed once the app starts listening.
type App struct {
	ContextProcessors []ContextProcessor
	ContextFinalizers []ContextFinalizer
	RecoverHandlers   []RecoverHandler

	// Logger to use when logging requests. By default, it's
	// gnd.la/log.Std, but you can set it to nil to avoid
	// logging at all and gain a bit more of performance.
	Logger *log.Logger

	// CookieOptions indicates the default options used
	// used for cookies. If nil, the default values as returned
	// by cookies.Defaults() are used.
	CookieOptions *cookies.Options
	// CookieCodec indicates the codec used for encoding and
	// decoding cookies. If nil, gob is used.
	CookieCodec *codec.Codec

	// Hasher is the hash function used to sign values. If nil,
	// it defaults to HMAC-SHA1.
	Hasher cryptoutil.Hasher

	// Cipherer is the cipher function used to encrypt values. If nil,
	// it defaults to AES.
	Cipherer cryptoutil.Cipherer

	// config received in New or defaultConfig, never nil
	cfg *Config

	handlers           []*handlerInfo
	trustXHeaders      bool
	appendSlash        bool
	errorHandler       ErrorHandler
	languageHandler    LanguageHandler
	name               string
	userFunc           UserFunc
	assetsManager      *assets.Manager
	templatesFS        vfs.VFS
	templatesMutex     sync.RWMutex
	templatesCache     map[string]*Template
	templateProcessors []TemplateProcessor
	namespace          *namespace
	templatePlugins    []*template.Plugin
	started            time.Time
	address            string
	mu                 sync.Mutex
	c                  *cache.Cache
	o                  *orm.Orm
	store              *blobstore.Blobstore
	kv                 kvs.KVS
	prepared           bool

	// Used for included apps
	included  []*includedApp
	parent    *App
	childInfo *includedApp
}

// Handle adds a new handler to the App. See the available HandlerOption functions
// and the HandlerOptions type for the available handler options.
//
// A named handler can be be reversed using Context.Reverse or
// the "reverse" template function. Use NamedHandler() to set a name.
//
// To add a host-specific Handler, use HostHandler().
func (app *App) Handle(pattern string, handler Handler, opts ...HandlerOption) {
	if handler == nil {
		panic(fmt.Errorf("handler for pattern %q can't be nil", pattern))
	}
	re := regexp.MustCompile(pattern)
	handlerOpts := HandlerOptions{}
	for _, v := range opts {
		handlerOpts = v(handlerOpts)
	}
	info := &handlerInfo{
		host:    handlerOpts.Host,
		name:    handlerOpts.Name,
		re:      re,
		rc:      newRegexpCache(re),
		handler: handler,
	}
	if p := literalRegexp(re); p != "" {
		info.path = p
		info.pathMatch = []int{0, len(p)}
	}
	app.handlers = append(app.handlers, info)
}

// HandleWebsocket has the same semantics as App.Handle, but responds to websocket
// requests rather than normal HTTP(S) requests.
func (app *App) HandleWebsocket(pattern string, handler WebsocketHandler, opts ...HandlerOption) {
	type wsKey int
	const (
		ctxKey wsKey = iota
	)
	if handler == nil {
		panic(fmt.Errorf("handler for websocket pattern %q can't be nil", pattern))
	}
	wsHandler := websocket.Handler(func(ws *websocket.Conn) {
		ctx := ws.Request().Context().Value(ctxKey).(*Context)
		handler(ctx, ws)
	})
	reqHandler := func(ctx *Context) {
		req := ctx.Request()
		newCtx := context.WithValue(req.Context(), ctxKey, ctx)
		newReq := ctx.Request().WithContext(newCtx)
		rw := ctx.ResponseWriter
		ctx.ResponseWriter = nil
		wsHandler.ServeHTTP(rw, newReq)
	}
	app.Handle(pattern, reqHandler, opts...)
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

// Once executes f with the first request it receives. This is mainly
// used for ORM initialization on App Engine.
func (app *App) Once(f func(*Context)) {
	// TODO: Make this more efficient, remove the
	// ContextProcessor once we've called f.
	var once sync.Once
	p := func(ctx *Context) bool {
		once.Do(func() {
			f(ctx)
		})
		return false
	}
	app.AddContextProcessor(p)
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

func (app *App) Include(prefix string, included *App, containerTemplate string) {
	if err := app.include(prefix, included, containerTemplate); err != nil {
		panic(err)
	}
	if app.namespace == nil {
		app.namespace = new(namespace)
	}
	if app.namespace.vars == nil {
		app.namespace.vars = make(map[string]interface{})
	}
	apps, _ := app.namespace.vars["Apps"].(map[string]interface{})
	if apps == nil {
		apps = make(map[string]interface{})
	}
	apps[included.name] = included
	app.namespace.vars["Apps"] = apps
}

func (app *App) include(prefix string, child *App, containerTemplate string) error {
	if child.parent != nil {
		return fmt.Errorf("app %v already has been included in another app", child)
	}
	if child.name == "" {
		return fmt.Errorf("included app %v can't have an empty name", child)
	}
	if prefix == "" {
		return fmt.Errorf("can't include app %s with empty prefix", child.name)
	}
	// prefix must start with / and end without /,
	// fix it if it doesn't match
	if prefix[0] != '/' {
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
	if containerTemplate == "" {
		return fmt.Errorf("empty container template while loading app %v", child.Name())
	}
	child.parent = app
	included := &includedApp{
		prefix:    prefix,
		app:       child,
		container: containerTemplate,
	}
	if containerTemplate != "" {
		// Check if the container is fine. Not completely sure if this
		// is a good idea, since the panic will happen on start rather
		// than on a request, but otherwise the error could go unnoticed
		// until a handler from the included app is invoked.
		if _, _, err := app.loadContainerTemplate(included); err != nil {
			return err
		}
	}
	child.childInfo = included
	if child.assetsManager != nil {
		if err := app.importAssets(included); err != nil {
			return fmt.Errorf("error importing %q assets: %s", child.name, err)
		}
	}
	for _, v := range child.templatePlugins {
		v.Template.AddNamespace(child.name)
		if err := app.rewriteAssets(v.Template, included); err != nil {
			return err
		}
		app.AddTemplatePlugin(v)
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
	// All checks passed, add the included app handler
	app.Handle("^"+prefix, includedAppHandler(child, prefix))
	return nil
}

// Parent returns the parent App. Note that this is non-nil only
// for apps which have been included into another app.
func (app *App) Parent() *App {
	return app.parent
}

// Included returns the apps which have been included by this
// app.
func (app *App) Included() []*App {
	apps := make([]*App, len(app.included))
	for ii, v := range app.included {
		apps[ii] = v.app
	}
	return apps
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
// a language handler it uses the language specified by DefaultLanguage().
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

// TemplatesFS returns the VFS for the templates assocciated
// with this app. By default, templates will be loaded from the
// tmpl directory relative to the application binary.
func (app *App) TemplatesFS() vfs.VFS {
	if app.templatesFS == nil {
		app.templatesFS = template.DefaultVFS()
	}
	return app.templatesFS
}

// SetTemplatesFS sets the VFS used to load the templates
// associated with this app. By default, templates will be loaded from the
// tmpl directory relative to the application binary.
func (app *App) SetTemplatesFS(fs vfs.VFS) {
	app.templatesFS = fs
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

// AddTemplatePlugin adds a *template.Plugin which will be added to
// all templates rendered by this app. See gnd.la/template.Plugin for
// more information.
func (app *App) AddTemplatePlugin(plugin *template.Plugin) {
	app.templatePlugins = append(app.templatePlugins, plugin)
}

// LoadTemplate loads a template using the template
// loader and the asset manager assocciated with
// this app
func (app *App) LoadTemplate(name string) (*Template, error) {
	app.templatesMutex.RLock()
	tmpl := app.templatesCache[name]
	app.templatesMutex.RUnlock()
	if tmpl == nil {
		var err error
		log.Debugf("Loading root template %s", name)
		if profile.On && profile.Profiling() {
			defer profile.Start("template").Note("load", name).End()
		}
		tmpl, err = app.loadTemplate(app.templatesFS, app.assetsManager, name)
		if err != nil {
			return nil, err
		}
		var funcs []*template.Func
		for _, v := range app.included {
			funcs = append(funcs, &template.Func{
				Name:   v.assetFuncName(),
				Fn:     v.assetFunc(tmpl.tmpl),
				Traits: template.FuncTraitPure,
			})
		}
		tmpl.tmpl.Funcs(funcs)
		for _, v := range app.templatePlugins {
			if err := tmpl.tmpl.AddPlugin(v); err != nil {
				return nil, fmt.Errorf("error adding plugin %q: %s", v.Template.Root(), err)
			}
		}
		if profile.On {
			if profilePlugin != nil {
				tmpl.tmpl.AddPlugin(profilePlugin)
			}
		}
		if err := tmpl.prepare(); err != nil {
			return nil, err
		}
		if !app.cfg.TemplateDebug {
			app.templatesMutex.Lock()
			if app.templatesCache == nil {
				app.templatesCache = make(map[string]*Template)
			}
			app.templatesCache[name] = tmpl
			app.templatesMutex.Unlock()
		}
	}
	return tmpl, nil
}

func (app *App) loadTemplate(fs vfs.VFS, manager *assets.Manager, name string) (*Template, error) {
	t := newTemplate(app, fs, manager)
	var vars map[string]interface{}
	if app.namespace != nil {
		var err error
		if vars, err = app.namespace.eval(nil); err != nil {
			return nil, err
		}
	}
	err := t.parse(name, vars)
	if err != nil {
		return nil, err
	}
	for _, v := range app.templateProcessors {
		t.tmpl, err = v(t.tmpl)
		if err != nil {
			return nil, err
		}
	}
	if app.parent != nil && fs == app.templatesFS {
		t.tmpl.AddNamespace(app.name)
		if !t.tmpl.IsFinal() {
			return app.parent.chainTemplate(t, app.childInfo)
		}
	}
	return t, nil
}

func (app *App) rewriteAssets(t *template.Template, included *includedApp) error {
	if !app.shouldImportAssets() {
		return nil
	}
	for _, group := range t.Assets() {
		for _, a := range group.Assets {
			if a.IsRemote() || a.IsHTML() {
				continue
			}
			orig := a.Name
			if a.IsTemplate() {
				orig = a.TemplateName()
			}
			name := included.renames[orig]
			if name == "" {
				return fmt.Errorf("asset %q referenced from template %q does not exist", a.Name, t.Name())
			}
			a.Rename(name)
		}
		group.Manager = app.assetsManager
	}
	fname := included.assetFuncName()
	for _, v := range t.Trees() {
		templateutil.WalkTree(v, func(n, p parse.Node) {
			if n.Type() == parse.NodeIdentifier {
				id := n.(*parse.IdentifierNode)
				if id.Ident == template.AssetFuncName {
					id.Ident = fname
				}
			}
		})
	}
	return nil
}

func (app *App) loadContainerTemplate(included *includedApp) (*Template, string, error) {
	container, err := app.loadTemplate(app.templatesFS, app.assetsManager, included.container)
	if err != nil {
		return nil, "", err
	}
	name := template.NamespacedName([]string{included.app.name}, "~")
	found := false
	var loc string
	for _, v := range container.tmpl.Trees() {
		if err != nil {
			return nil, "", err
		}
		templateutil.WalkTree(v, func(n, p parse.Node) {
			if err != nil {
				return
			}
			if templateutil.IsPseudoFunction(n, "app") {
				if found {
					dloc, _ := v.ErrorContext(n)
					err = fmt.Errorf("duplicate {{ app }} node in container template %q: %s and %s",
						included.container, loc, dloc)
					return
				}
				// Used for error message if duplicate is found
				loc, _ = v.ErrorContext(n)
				found = true
				tmpl := templateutil.TemplateNode(name, n.Position())
				err = templateutil.ReplaceNode(n, p, tmpl)
			}
		})
	}
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "", fmt.Errorf("container template %q does not contain an {{ app }} node", included.container)
	}
	return container, name, nil
}

func (app *App) chainTemplate(t *Template, included *includedApp) (*Template, error) {
	log.Debugf("chaining template %s", t.tmpl.Name())
	container, name, err := app.loadContainerTemplate(included)
	if err != nil {
		return nil, err
	}
	if err := app.rewriteAssets(t.tmpl, included); err != nil {
		return nil, err
	}
	if err := container.tmpl.InsertTemplate(t.tmpl, name); err != nil {
		return nil, err
	}
	return container, nil
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

// HandleAssets adds several handlers to the app which handle
// assets efficiently and allows the use of the "assset"
// function from the templates. This function will also modify the
// asset loader associated with this app. prefix might be a relative
// (e.g. /static/) or absolute (e.g. http://static.example.com/) url
// while dir should be the path to the directory where the static
// assets reside. You probably want to use pathutil.Relative()
// to define the directory relative to the application binary. Note
// that /favicon.ico and /robots.txt will be handled too, but they
// will must be in the directory which contains the rest of the assets.
func (app *App) HandleAssets(prefix string, dir string) {
	fs, err := vfs.FS(dir)
	if err != nil {
		panic(err)
	}
	manager := assets.New(fs, prefix)
	app.SetAssetsManager(manager)
	app.addAssetsManager(manager, true)
}

func (app *App) addAssetsManager(manager *assets.Manager, main bool) {
	handler := HandlerFromHTTPFunc(manager.Handler())
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
		return "", errors.New("can't reverse, no handler name specified")
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
			reversed, err := formatRegexp(v.rc, args)
			if err != nil {
				if acerr, ok := err.(*argumentCountError); ok {
					if acerr.Min == acerr.Max {
						return true, "", fmt.Errorf("handler %q requires exactly %d arguments, %d received instead",
							name, acerr.Min, len(args))
					}
					return true, "", fmt.Errorf("handler %q requires at least %d arguments and at most %d arguments, %d received instead",
						name, acerr.Min, acerr.Max, len(args))
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
// port (see Address() and Port).
func (app *App) ListenAndServe() error {
	if err := app.Prepare(); err != nil {
		return err
	}
	if err := app.checkPort(); err != nil {
		return err
	}
	signal.Emit(WILL_LISTEN, app)
	app.started = time.Now().UTC()
	if devserver.IsActive() {
		// Attach the automatic reload template plugin to automatically
		// reload the page when the server restarts
		app.AddTemplateVars(devserver.TemplateVars(&Context{}))
		app.AddTemplatePlugin(devserver.ReloadPlugin())
	} else {
		if app.Logger != nil {
			if app.address != "" {
				app.Logger.Infof("Listening on %s, port %d", app.address, app.cfg.Port)
			} else {
				app.Logger.Infof("Listening on port %d", app.cfg.Port)
			}
		}
	}
	var err error
	time.AfterFunc(500*time.Millisecond, func() {
		if err == nil {
			signal.Emit(DID_LISTEN, app)
		}
	})
	err = http.ListenAndServe(app.address+":"+strconv.Itoa(app.cfg.Port), app)
	return err
}

// MustListenAndServe works like ListenAndServe, but panics if
// there's an error
func (app *App) MustListenAndServe() {
	err := app.ListenAndServe()
	if err != nil {
		log.Panicf("error listening on port %d: %s", app.cfg.Port, err)
	}
}

// Cache returns this app's cache connection, using
// DefaultCache(). Use gnd.la/config to change the default
// cache. The cache.Cache is initialized only once and shared
// among all requests and tasks served from this app.
// On App Engine, this method always returns an error. Use
// Context.Cache instead.
func (app *App) Cache() (*cache.Cache, error) {
	return app.cache()
}

// App returns this app's ORM connection, using the
// default database parameters, as returned by DefaultDatabase().
// Use gnd.la/config to change the default ORM. The orm.Orm
// is initialized only once and shared among all requests and
// tasks served from this app. On App Engine, this method always
// returns an error. Use Context.Orm instead. To perform ORM
// initialization, use App.Once.
func (app *App) Orm() (*orm.Orm, error) {
	return app.orm()
}

// prepareOrm must be called only in App instances without a
// parent. If it doesn't fail, it sets the o field in the App.
func (app *App) prepareOrm() error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.o != nil {
		return nil
	}
	if app.parent != nil {
		var err error
		app.o, err = app.parent.orm()
		return err
	}
	o, err := app.openOrm()
	if err != nil {
		return err
	}
	if err := o.Initialize(); err != nil {
		o.Close()
		return err
	}
	app.o = o
	return nil
}

func (app *App) openOrm() (*orm.Orm, error) {
	if app.parent != nil {
		return app.parent.openOrm()
	}
	db := app.cfg.Database
	if db == nil {
		return nil, errNoDefaultDatabase
	}
	o, err := orm.New(db)
	if err != nil {
		return nil, err
	}
	if app.Logger != nil && app.Logger.Level() == log.LDebug {
		o.SetLogger(app.Logger)
	}
	return o, nil
}

// Blobstore returns a blobstore using the default blobstore
// parameters, as returned by DefaultBlobstore(). Use
// gnd.la/config to change the default blobstore. See
// gnd.la/blobstore for further information on using the blobstore.
// Note that this function does not work on App Engine. Use Context.Blobstore
// instead.
func (app *App) Blobstore() (*blobstore.Blobstore, error) {
	return app.blobstore()
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
	ctx.statusCode = -code
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
	} else {
		buf.WriteString("Panic")
	}
	elapsed := ctx.Elapsed()
	fmt.Fprintf(&buf, " (after %s): %v\n", elapsed, err)
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
			req = stringutil.Lines(string(dump), 0, 10000, true)
			buf.WriteString("\nRequest:\n")
			buf.WriteString(req)
		}
		// Check if there are any attached files that we might
		// want to send in an email
		if !app.cfg.Debug && mail.AdminEmail() != "" {
			ctx.R.ParseMultipartForm(32 << 20) // 32 MiB, as stdlib
			if form := ctx.R.MultipartForm; form != nil {
				var count int
				var attachments []*mail.Attachment
				var message bytes.Buffer
				if len(form.File) > 0 {
					for k, v := range form.File {
						for _, file := range v {
							f, err := file.Open()
							if err != nil {
								fmt.Fprintf(&message, "%s => error %s", k, err)
								continue
							}
							attachment, err := mail.NewAttachment(file.Filename, f)
							attachment.ContentType = file.Header.Get("Content-Type")
							f.Close()
							if err != nil {
								fmt.Fprintf(&message, "%s => error %s", k, err)
								continue
							}
							count++
							fmt.Fprintf(&message, "%s => %s (%s)", k, attachment.Name, attachment.ContentType)
							attachments = append(attachments, attachment)
						}
					}
					fmt.Fprintf(&message, "\nError:\n%s", buf.String())
					host, _ := os.Hostname()
					from := mail.DefaultFrom()
					if from == "" {
						from = fmt.Sprintf("errors@%s", host)
					}
					msg := &mail.Message{
						From:        from,
						To:          mail.Admin,
						Subject:     fmt.Sprintf("Panic with %d attached files on %s", count, host),
						TextBody:    message.String(),
						Attachments: attachments,
					}
					ctx.SendMail("", nil, msg)
				}
			}
		}
	}
	ctx.Logger().Error(buf.String())
	if app.cfg.Debug {
		app.errorPage(ctx, elapsed, skip, stackSkip, req, err)
	} else {
		app.handleHTTPError(ctx, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (app *App) errorPage(ctx *Context, elapsed time.Duration, skip int, stackSkip int, req string, err interface{}) {
	t := newInternalTemplate(app)
	if terr := t.parse("panic.html", devserver.TemplateVars(&Context{})); terr != nil {
		panic(terr)
	}
	if devserver.IsActive() {
		t.tmpl.AddPlugin(devserver.ReloadPlugin())
	}
	if terr := t.prepare(); terr != nil {
		panic(terr)
	}
	stack := runtimeutil.FormatStackHTML(stackSkip + 1)
	location, code := runtimeutil.FormatCallerHTML(skip+1, 5, true, true)
	ctx.statusCode = -http.StatusInternalServerError
	data := map[string]interface{}{
		"Error":       fmt.Sprintf("%v", err),
		"Subtitle":    fmt.Sprintf("(after %s)", elapsed),
		"Location":    location,
		"Code":        code,
		"Stack":       stack,
		"Request":     req,
		"Name":        filepath.Base(os.Args[0]),
		"IsDevServer": devserver.IsDevServer(app),
	}
	if err := t.Execute(ctx, data); err != nil {
		var msg string
		if file, line, ok := runtimeutil.PanicLocation(); ok {
			msg = fmt.Sprintf("error rendering error page: %v @ %s:%d)", err, file, line)
		} else {
			msg = fmt.Sprintf("error rendering error page: %v", err)
		}
		ctx.WriteString(msg)
	}
}

// ServeHTTP is called from the net/http system. You shouldn't need
// to call this function
func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := app.newContext(w, r)
	if profile.On && shouldProfile(ctx) {
		profile.Begin()
		defer profile.End(0)
	}
	defer app.closeContext(ctx)
	defer app.recover(ctx)
	if app.runProcessors(ctx) {
		return
	}
	app.serveOrNotFound(r.URL.Path, ctx)
}

func (app *App) serveOrNotFound(path string, ctx *Context) {
	if !app.serve(path, ctx) {
		// Not Found
		app.handleHTTPError(ctx, "Not Found", http.StatusNotFound)
	}
}

func (app *App) serve(path string, ctx *Context) bool {
	if handler := app.matchHandler(path, ctx); handler != nil {
		handler(ctx)
		return true
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
	p := &regexpProvider{}
	ctx := &Context{R: r, ResponseWriter: w, app: app, provider: p, reProvider: p, started: time.Now()}
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
	if !ctx.background && app.Logger != nil && ctx.R != nil && ctx.R.URL.Path != monitorAPIPage {
		// Log at most with Warning level, to avoid potentially generating
		// an email to the admin when running in production mode. If there
		// was an error while processing this request, it has been already
		// emailed to the admin, along the stack trace, in recover().
		logger := ctx.Logger()
		message := strings.Join([]string{ctx.R.Method, ctx.R.RequestURI, ctx.RemoteAddress(),
			strconv.Itoa(ctx.statusCode), ctx.Elapsed().String()}, " ")
		if ctx.statusCode >= 400 {
			logger.Warning(message)
		} else {
			logger.Info(message)
		}
	}

}

// closeContext calls CloseContexts and stores the context in
// in the pool for reusing it.
func (app *App) closeContext(ctx *Context) {
	app.CloseContext(ctx)
}

func (app *App) shouldImportAssets() bool {
	return !app.cfg.TemplateDebug || internal.InAppEngine()
}

func (app *App) importAssets(included *includedApp) error {
	im := included.app.assetsManager
	if !app.shouldImportAssets() {
		im.SetPrefix(included.prefix + im.Prefix())
		return nil
	}
	m := app.assetsManager
	prefix := strings.ToLower(included.app.name)
	renames := make(map[string]string)
	err := vfs.Walk(im.VFS(), "/", func(fs vfs.VFS, p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if p != "" && p[0] == '/' {
			p = p[1:]
		}
		log.Debugf("will import asset %v from app %s", p, included.app.name)
		src, err := im.Load(p)
		if err != nil {
			return err
		}
		defer src.Close()
		seeker, err := assets.Seeker(src)
		if err != nil {
			return err
		}
		sum := hashutil.Fnv32a(seeker)
		nonExt := p[:len(p)-len(path.Ext(p))]
		dest := path.Join(prefix, nonExt+".gen."+sum+path.Ext(p))
		renames[p] = dest
		log.Debugf("importing asset %q as %q", p, dest)
		if m.Has(dest) {
			return nil
		}
		f, err := m.Create(dest, true)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := seeker.Seek(0, os.SEEK_SET); err != nil {
			return err
		}
		if _, err := io.Copy(f, seeker); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	included.renames = renames
	return nil
}

func (app *App) locked(f func()) {
	app.mu.Lock()
	defer app.mu.Unlock()
	f()
}

// Signer returns a *cryptoutil.Signer using the given salt and
// the App Hasher and Secret to sign values. If salt is smaller
// than 16 bytes or the App has no Secret, an error is returned.
func (app *App) Signer(salt []byte) (*cryptoutil.Signer, error) {
	if len(salt) < 16 {
		return nil, fmt.Errorf("salt must be at least 16 bytes, it's %d", len(salt))
	}
	secret := app.cfg.Secret
	if secret == "" {
		return nil, errNoSecret
	}
	return &cryptoutil.Signer{
		Hasher: app.Hasher,
		Salt:   salt,
		Key:    []byte(secret),
	}, nil
}

// Encrypter returns a *cryptoutil.Encrypter using the App
// Cipherer and Key to encrypt values. If the App has no
// Key, an error will be returned.
func (app *App) Encrypter() (*cryptoutil.Encrypter, error) {
	key := app.cfg.EncryptionKey
	if key == "" {
		return nil, errNoKey
	}
	return &cryptoutil.Encrypter{
		Cipherer: app.Cipherer,
		Key:      []byte(key),
	}, nil
}

// EncryptSigner returns a *cryptoutil.EncryptSigner composed by
// App.Signer and App.Encrypter. See those methods for more details.
func (app *App) EncryptSigner(salt []byte) (*cryptoutil.EncryptSigner, error) {
	signer, err := app.Signer(salt)
	if err != nil {
		return nil, err
	}
	encrypter, err := app.Encrypter()
	if err != nil {
		return nil, err
	}
	return &cryptoutil.EncryptSigner{
		Encrypter: encrypter,
		Signer:    signer,
	}, nil
}

// Clone returns a copy of the *App. This is mainly useful for including an
// app multiple times. Note that cloning an App which has been already included
// is considered a programming error and will result in a panic.
func (app *App) Clone() *App {
	if app.parent != nil {
		panic(fmt.Errorf("can't clone app %s, it has been already included", app.name))
	}
	a := *app
	a.kv = *(a.kv.Copy())
	return &a
}

// Prepare is automatically called for you. This function is
// only exposed because the gnd.la/app/tester package needs
// to call it to set the App up without making it listen on
// a port.
func (app *App) Prepare() error {
	// Make Prepare() idempotent. Otherwise the tester package
	// would need a lot of extra logic to deal with GAE vs
	// non-GAE environments.
	if app.prepared {
		return nil
	}
	// Initialize the ORM first, so admin commands
	// run with the ORM ready to be used.
	if app.parent == nil {
		err := app.prepareOrm()
		if err != nil && err != errNoDefaultDatabase && err != errNoAppOrm {
			return err
		}
	}
	signal.Emit(WILL_PREPARE, app)
	if s := app.cfg.Secret; s != "" && len(s) < 32 &&
		os.Getenv("GONDOLA_ALLOW_SHORT_SECRET") == "" && !devserver.IsDevServer(app) {
		return fmt.Errorf("secret %q is too short, must be at least 32 characters - use gondola random-string to generate one", s)
	}
	for _, v := range app.included {
		child := v.app
		child.cfg = app.cfg
		child.CookieOptions = app.CookieOptions
		child.CookieCodec = app.CookieCodec
		child.Hasher = app.Hasher
		child.Cipherer = app.Cipherer
		child.languageHandler = app.languageHandler
		child.userFunc = app.userFunc
		child.Logger = app.Logger
	}
	// Add template plugins from each included app to all the other apps
	for _, h := range app.templatePlugins {
		if h.Position == assets.None {
			continue
		}
		for _, v := range app.included {
			child := v.app
			has := false
			for _, ch := range child.templatePlugins {
				if ch == h {
					has = true
					break
				}
			}
			if !has {
				child.templatePlugins = append(child.templatePlugins, h)
			}
		}
	}
	app.prepared = true
	signal.Emit(DID_PREPARE, app)
	return nil
}

// Config returns the App configuration, which will always
// be non-nil. Note that is not recommended altering the Config
// fields from your code. Instead, use the configuration file and
// flags.
func (app *App) Config() *Config {
	return app.cfg
}

// Transform transforms all the registered handlers using the given
// Transformer. Note that handlers registered after this call won't
// be transformed. This function might be used to cache an entire
// App using gnd.la/cache/layer.
func (app *App) Transform(tr Transformer) {
	for _, v := range app.handlers {
		v.handler = tr(v.handler)
	}
}

// New returns a new App initialized with the default config.
func New() *App {
	return NewWithConfig(nil)
}

// NewWithConfig returns a new App initialized with the given config. If config
// is nil, the default configuration (exposed via gnd.la/config) is
// used instead.
func NewWithConfig(config *Config) *App {
	cfg := config
	if cfg == nil {
		// Make a copy of the configuration
		cc := defaultConfig
		cfg = &cc
	}
	a := &App{
		Logger:         log.Std,
		cfg:            cfg,
		appendSlash:    true,
		templatesCache: make(map[string]*Template),
	}
	// Used to automatically reload the page on panics when the server
	// is restarted.
	if cfg.Debug || profile.On {
		a.Handle("^/debug/pprof/cmdline", wrap(pprof.Cmdline))
		a.Handle("^/debug/pprof/profile", wrap(pprof.Profile))
		a.Handle("^/debug/pprof", wrap(pprof.Index))
		a.Handle("^/debug/profile", profileInfoHandler)
		a.Handle(monitorAPIPage, monitorAPIHandler)
		a.Handle(monitorPage, monitorHandler)
		a.addAssetsManager(internalAssetsManager, false)
	}
	return a
}

func wrap(f func(w http.ResponseWriter, r *http.Request)) Handler {
	return func(ctx *Context) {
		f(ctx, ctx.R)
	}
}
