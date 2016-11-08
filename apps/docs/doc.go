// Package docs implements a Gondola application for browsing package
// documentation.
//
// This application requires a Go source code checkout to be present
// on the server as well as GOROOT and GOPATH properly configured.
// Additionaly, a working Go installation is required to automatically
// download and update packages.
//
// To configure the packages to list in the index, use DocsApp.Groups.
//
// This application can also automatically fetch and update the packages
// listed in the index. See StartUpdatingPackages and StopUpdatingPackages.
package docs
