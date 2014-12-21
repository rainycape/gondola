// Package bootstrap implements some helper functions intended
// to be used with the Bootstrap front-end framework.
// See http://twitter.github.io/bootstrap/ for more details.
//
// This package defines the "bootstrap" asset, which serves bootstrap
// from http://www.bootstrapcdn.com. It receives a single argument with the
// desired bootstrap version. Only version 3 is supported. e.g.
//
//  bootstrap: 3.0.0
//
// This asset also supports the following options:
//
//  nojs (bool): disables loading bootstrap's javascript library
//  e.g. bootstrap|nojs: 2.3.2
//
// See gnd.la/template and gnd.la/template/assets for more information
// about template functions and the assets pipeline.
//
// Importing this package will also register FormRenderer as the default
// gnd.la/form renderer and PaginatorRenderer as the default
// gnd.la/html/paginator renderer.
package bootstrap
