// Package datastore implements an App Engine datastore driver
// the Gondola's ORM.
//
// This package is only available when building your application using
// the App Engine Go SDK. To enable the driver, import its package:
//
//  import (
//      _ "gnd.la/orm/driver/datastore"
//  )
//
// Some caveats your need to be aware of:
//
//  - The datastore driver does not support OR nor NEQ queries.
//  - The datastore driver is not relational (no support for foreign keys nor JOINs).
//  - While auto_increment its supported, the numeric IDs won't be sequential, only
//      strictly increasing (i.e. IDs will always increase, but there might be gaps
//      between them).
package datastore
