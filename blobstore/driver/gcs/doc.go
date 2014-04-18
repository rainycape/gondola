// Package gcs provides a Google Cloud Storage driver for the Blobstore.
//
// Note that currently this package only works on App Engine.
// To enable this driver, import it in your application:
//
//  import (
//      _ "gnd.la/blobstore/driver/gcs"
//  )
//
// The URL format for this driver is:
//
//  gcs://bucket_name
//
// Is the bucket is omitted, the default (your-app-id.appspot.com) is used.
// The bucket must already exist.
package gcs
