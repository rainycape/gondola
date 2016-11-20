package reusableapp

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"gnd.la/app"
	"gnd.la/template/assets"
	"gnd.la/util/vfsutil"
)

// App allows implementing apps which can be directly included
// in a gnd.la/app.App. Use New to create an App.
type App struct {
	app.App
	name                  string
	Prefix                string
	ContainerTemplateName string
	opts                  *Options
	// The directory for the source file which called New(), used
	// for the baked data.
	dir      string
	attached bool
}

func (a *App) SetName(name string) {
	if a.attached {
		panic(fmt.Errorf("can't rename reusable app %v, it's already attached", a.name))
	}
	a.name = name
}

func (a *App) mustOpenVFS(defaultRel string, optRel string, data string) (fs vfsutil.VFS, rel string) {
	rel = defaultRel
	if optRel != "" {
		rel = optRel
	}
	abs := a.relativePath(rel)
	if dirExists(abs) || optRel != "" {
		fs, err := openVFS(a.name, abs, data)
		if err != nil {
			panic(err)
		}
		return fs, rel
	}
	return nil, ""
}

func (a *App) relativePath(rel string) string {
	if a.dir == "" {
		panic(fmt.Errorf("could not determine relative path for assets from app %s", a.name))
	}
	return filepath.Join(a.dir, filepath.FromSlash(rel))
}

// Attach attaches a reusable app into its parent app.
func (a *App) Attach(parent *app.App) {
	parent.Include(a.Prefix, a.name, &a.App, a.ContainerTemplateName)
}

// Data returns the Data field from the Options struct, as passed in to New.
func (a *App) Data() interface{} {
	return a.opts.Data
}

// New returns a new App. Any errors will result in a panic, Since
// this function should be called only during app initialization.
func New(opts Options) *App {
	if opts.Name == "" {
		panic(errors.New("reusable app name can't be empty"))
	}
	a := &App{
		App:  *app.New(),
		opts: &opts,
		name: opts.Name,
	}
	var k interface{} = reusableAppKey
	if opts.DataKey != nil {
		k = opts.DataKey
	}
	a.Set(k, a)
	_, file, _, ok := runtime.Caller(1)
	if ok {
		a.dir = filepath.Dir(file)
	}
	assetsFS, assetsRel := a.mustOpenVFS("assets", opts.AssetsDir, opts.AssetsData)
	if assetsFS != nil {
		assetsPrefix := path.Clean("/" + filepath.ToSlash(assetsRel) + "/")
		manager := assets.New(assetsFS, assetsPrefix)
		a.SetAssetsManager(manager)
		a.Handle("^"+assetsPrefix, app.HandlerFromHTTPFunc(manager.Handler()))
	}
	templatesFS, _ := a.mustOpenVFS("tmpl", opts.TemplatesDir, opts.TemplatesData)
	if templatesFS != nil {
		a.SetTemplatesFS(templatesFS)
	}
	return a
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

type key int

const (
	reusableAppKey key = iota
)

// Data is a shorthand for AppData(ctx.App())
func Data(ctx *app.Context) interface{} {
	return AppData(ctx.App())
}

// AppData returns Options.Data from the Options type, as passed to New.
func AppData(a *app.App) interface{} {
	return AppDataWithKey(a, reusableAppKey)
}

// AppDataWithKey works similarly to AppData, but uses the provided key instead.
// Also, if the data is not found in the *app.App passed in as the first argument,
// its children apps are also searched. This allows reusable apps to retrieve their
// additional data in contexts where the reusable app pointer is not available.
// (e.g. in template plugins which are called from the parent app). See gnd.la/apps/users
// for an example of this usage.
func AppDataWithKey(a *app.App, key interface{}) interface{} {
	ra, _ := a.Get(key).(*App)
	if ra != nil {
		return ra.Data()
	}
	if key != reusableAppKey {
		for _, ia := range a.Included() {
			if data := AppDataWithKey(ia, key); data != nil {
				return data
			}
		}
	}
	return nil
}
