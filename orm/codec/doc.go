// Package codec includes the interfaces and functions for implementing
// and using ORM field codecs, which enable objects to be automatically
// encoded, when saving them to the database, and decoded, when loading
// them from the database.
//
// Codecs for "json" and "gob" are already provided by this package, but
// users can define their own ones. See the Codec interface and the
// Register function.
package codec
