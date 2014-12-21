// Package bootstrap implements some helper functions intended
// to be used with the Bootstrap front-end framework.
// See http://twitter.github.io/bootstrap/ for more details.
//
// Importing this package has also some side-effects related to
// templates. It defines the following template functions:
//
//  fa <string>: returns the font awesome 4 (and hopefully future versions) icon named by string
//  e.g. {{ fa "external-link" } => <i class="fa fa-external-link"></i>
//
//  fa3 <string>: returns the font awesome 3 icon named by string
//  e.g. {{ fa3 "external-link" } => <i class="icon-external-link"></i>
//
// It also adds parsers for the "bootstrap" asset key, which serves boostrap
// from http://www.bootstrapcdn.com. It receives a single argument with the
// desired bootstrap version. Both 2.x and 3.x are supported. e.g.
//
//  bootstrap: 3.0.0
//
// This asset also supports the following options:
//
//  fontawesome (string): load also the specified Font Awesome version
//  e.g.: bootstrap|fontawesome=4.0.3: 3.0.2
//
//  nojs (bool): disables loading bootstrap's javascript library
//  e.g. bootstrap|nojs: 2.3.2
//
// See gnd.la/template and gnd.la/template/assets for more information
// about template functions and the assets pipeline.
package bootstrap
