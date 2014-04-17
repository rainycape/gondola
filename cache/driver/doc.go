// Package driver includes the interfaces required to implement
// a Gondola cache driver, as well as the dummy, memory and file drivers.
//
// Additionally, drivers with no external dependencies are provided
// by this package. Note that Gondola already imports this package, so
// users don't need to explicitely import it.
//
// The provided drivers are:
//
//  - dummy:// - a dummy driver which does not cache data, useful for development
//  - memory://[#max_size={size} - a memory driver with an optional maximum size
//  - file://path[#max_size={size} a file based driver with an optional maximum size
//
// Sizes admit the K, M, G and T suffixes to represent Kilobytes, Megabytes, Gigabytes and
// Terabytes, respectivelly. When there's no prefix, the value is assumed to be in bytes. Note
// that real numbers can be used, like e.g. 1.5G
//
// Paths which don't start with a / are interepreted as relative to the application binary
// (using gnd.la/util/pathutil.Relative), while paths starting by / are interpreted as absolute.
// Note that paths should aways use forward slashes, even in platforms which use the backslash
// by default (e.g. /C:/Documents/my_cache_path).
package driver
