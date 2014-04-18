// Package cache implements a caching system with pluggable backends.
//
// The cache is configured with a gnd.la/config.URL, which takes the form:
//
//  scheme://value?var=value&var2=value#anothervar=anothervalue
//
// While each driver might implement its own options, there are 3 options
// which apply to all drivers and are specified after the # character. They are:
//
//  - codec: The codec used for encoding/decoding the cached objects. See gnd.la/encoding/codec for the available ones.
//  - pipe: A pipe to pass the data trough, usually for compressing it. See gnd.la/encoding/pipe for the available ones.
//  - prefix: A prefix to be prepended to all keys stored.
//
// Note that these options are not mandatory. For the available drivers, see gnd.la/cache/driver for the ones without
// dependencies and its subpackages for the ones with external dependencies.
//
// Some examples of valid configurations:
//
//  memcache://localhost#codec=json&pipe=zlib
//  memory://#max_size=1.5G
//  file://cache#max_size=512M
package cache
