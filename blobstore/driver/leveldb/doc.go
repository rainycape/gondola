// Package leveldb implements the levelb driver for
// the blobstore.
//
// This package is still experimental and should not be used
// in production.
//
// This driver uses the part immediately after the leveldb://
// as the root directory for two leveldb databases, called files
// and chunks. Note that the path might be either absolute or
// relative (in the latter case is interpreted as relative to
// the application binary).
// Some examples:
//
//  leveldb:///var/data/files - absolute path
//  leveldb://storage - relative path, files are stored in the storage dir relative to the binary
package leveldb
