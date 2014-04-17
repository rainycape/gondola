// Package memcache implements a Gondola cache driver using memcache.
//
// This package works both on standalone Go installations and Google App
// Engine. The URL format for this driver is:
//
//  memcache://host1[:port][,host2][,hostn][#timeout={seconds}&max_idle={max}
//
// If no port is provided, memcached's default is used.
// If no timeout is provided, 200ms is used as a default. Setting timeout
// to zero disables timeouts. max_idle represents the maximum number of idle
// connections kept per host. The default is 2.
package memcache
