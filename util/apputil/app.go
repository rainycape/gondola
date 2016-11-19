package apputil

import (
	"errors"

	"gnd.la/app"
	"gnd.la/util/vfsutil"
)

// ReusableApp allows implementing apps which can be directly included
// in a gnd.la/app.App. Use NewReusableApp to create a ReusableApp.
type ReusableApp struct {
	*app.App
	Prefix       string
	BaseTemplate string
}

// OpenVFS opens a VFS for a reusable app. Since reusable apps need to have
// their non-go files baked, they should use this function for opening their
// assets and templates. This function will initialize the VFS with the baked
// resources when the following conditions are met:
//
// - baked is non-empty
// - the environment variable GONDOLA_APP_NO_VFS doesn't contain the app name (as passed to NewReusableApp)
//
// Otherwise, a VFS pointing the the directory indicated by relativePath (relative to
// the source file calling this funciton) will be returned. This is useful while writing
// reusable apps, so you can start the development server with:
//
//	GONDOLA_APP_NO_VFS=MyApp
//
// And then OpenVFS will use the assets from the filesystem. Once you're ready to
// commit the changes, you can regenerate the baked assets via gondola bake
// (usually called from go generate) and users of your app will be able to just
// go get it.
func (a *ReusableApp) OpenVFS(relativePath string, baked string) (vfsutil.VFS, error) {
	return openVFS(a.Name(), relativePath, baked)
}

// MustOpenVFS is a shorthand for OpenVFS which panics in case of error.
func (a *ReusableApp) MustOpenVFS(relativePath string, baked string) vfsutil.VFS {
	vfs, err := a.OpenVFS(relativePath, baked)
	if err != nil {
		panic(err)
	}
	return vfs
}

// NewReusableApp returns a new ReusableApp. The name argument must be
// non-empty, otherwise this function will panic.
func NewReusableApp(name string) *ReusableApp {
	if name == "" {
		panic(errors.New("reusable app name can't be empty"))
	}
	a := app.New()
	a.SetName(name)
	return &ReusableApp{
		App: a,
	}
}
