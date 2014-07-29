// Package form implements a form renderer using Bootstrap.
//
// Importing this package will register Renderer as the default
// form renderer, so users should usually import this package only
// for its side-effects and use gnd.la/form directly.
//
//  import (
//	_ "gnd.la/bootstrap/form"
//  )
package form

import "gnd.la/form"

func newRenderer() form.Renderer {
	return &Renderer{}
}

func init() {
	form.SetDefaultRenderer(newRenderer)
}
