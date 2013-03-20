package config

// This package exist only to allow
// gondola/mux to set configuration options
// for gondola/template and it should not be
// used from user code. Use instead the methods
// in gondola/template or, even better, use
// the methods in gondola/mux and let it configure
// the templates

import (
	"gondola/util"
)

var (
	staticFilesUrl string
	templatesPath  = util.RelativePath("tmpl")
)

func StaticFilesUrl() string {
	return staticFilesUrl
}

func SetStaticFilesUrl(url string) {
	staticFilesUrl = url
}

func Path() string {
	return templatesPath
}

func SetPath(p string) {
	templatesPath = p
}
