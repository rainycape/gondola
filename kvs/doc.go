// Package kvs implements a generic container for assocciating keys with values
// and easily obtaining type safe-functions for setting and retrieving them.
//
// For getting and setting multiple values of the same type associated with a
// container, use Funcs. If you only want to store a single value per type,
// use TypeFuncs instead, which will always use a key derived from the return
// type.
//
// A typical usage of this package looks like:
//
//      var (
//          getter func(kvs.Storage, key interface{}) V
//          setter func(kvs.Storage, key interface{}, value V)
//      )
//
//      func Get(kvs.Storage s, key interface{}) V {
//          return getter(kvs, key)
//      }
//
//      func Set(kvs.Storage s, key interface{}, value V) {
//          return setter(s, key, value)
//      }
//
//      func init() {
//          kvs.Funcs(&getter, &setter)
//      }
//
//
// Note that this function will panic if the prototypes of the functions don't,
// match the expected ones.
package kvs
