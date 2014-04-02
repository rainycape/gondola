// Package file implements the file driver for
// the blobstore.
//
// This driver uses the part immediately after the file://
// as the root directory for storing the files. Note that
// the path might be either absolute or relative (in the latter
// case is interpreted as relative to the application binary).
// Some examples:
//
//  file:///var/data/files - absolute path
//  file://storage - relative path, files are stored in the storage dir relative to the binary
package file
