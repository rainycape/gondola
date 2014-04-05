// Package blobstore implements a blob storage system with
// pluggable backends.
//
// In most cases, users will want to use the gnd.la/app.App.Blobstore
// and gnd.la/app.Context.Blobstore helper methods to obtain a connection
// to the default blobstore. The default blobstore can be set using
// gnd.la/config.
//
// There might be additional considerations for the backend you want to use.
// Please, see this package's subpackages for the available backends and the
// documentation about them.
//
// File metadata must be a struct and is serialized using BSON. For more
// information about the BSON format and struct tags that you might use to
// control the serialization, see gnd.la/internal/bson.
package blobstore
