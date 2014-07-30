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
//
// To create the default bucket from a GAE project without billing enabled, open
// https://appengine.google.com, select your project, then go to Application
// Settings in the left sidebar and scroll to the bottom. There's a section
// labeled "Cloud Integration" just at the end with a "Create" button. Click
// on it and the default GCS bucket will be created.
package gcs
