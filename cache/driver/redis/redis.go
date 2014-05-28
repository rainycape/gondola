// Package redis implements a Gondola cache driver using redis.
//
// The URL format for this driver is:
//
//  - redis://host[:port][#password={pw}&db={number}&max_idle={number}&max_active={number}&idle_timeout={seconds}]
//
// If no db is provided, it defaults to -1.
// For the defaults and the explanation for the rest of the parameters,
// see DefaultMaxIdle, DefaultMaxActive and DefaultIdleTimeout.
package redis

import (
	"fmt"
	"time"

	"gnd.la/cache/driver"
	"gnd.la/config"

	"github.com/garyburd/redigo/redis"
)

const (
	// DefaultMaxIdle is the maximum number of idle connections
	// kept in the connection pool.
	DefaultMaxIdle = 8
	// DefaultMaxActive is the maximum number of connections that
	// will be open at any given time. Setting it to zero, the
	// default value, won't limit the number of connections.
	DefaultMaxActive = 0
	// DefaultIdleTimeout is the amount of seconds after an idle
	// connection will be dropped from the pool.
	DefaultIdleTimeout = 300
)

type redisDriver struct {
	pool *redis.Pool
}

func (r *redisDriver) Set(key string, b []byte, timeout int) error {
	conn := r.pool.Get()
	var err error
	if timeout == 0 {
		_, err = conn.Do("SET", key, b)
	} else {
		_, err = conn.Do("SETEX", key, int32(timeout), b)
	}
	conn.Close()
	return err
}

func (r *redisDriver) Get(key string) ([]byte, error) {
	conn := r.pool.Get()
	reply, err := conn.Do("GET", key)
	conn.Close()
	if err != nil {
		return nil, err
	}
	switch reply := reply.(type) {
	case []byte:
		return reply, nil
	case string:
		return []byte(reply), nil
	case redis.Error:
		return nil, reply
	}
	// nil returned, item was not present
	return nil, nil
}

func (r *redisDriver) GetMulti(keys []string) (map[string][]byte, error) {
	args := make([]interface{}, len(keys))
	for ii, v := range keys {
		args[ii] = v
	}
	conn := r.pool.Get()
	reply, err := conn.Do("MGET", args...)
	conn.Close()
	if err != nil {
		return nil, err
	}
	if e, ok := reply.(redis.Error); ok {
		return nil, e
	}
	values := reply.([]interface{})
	ret := make(map[string][]byte, len(keys))
	for ii, v := range values {
		if v != nil {
			if b, ok := v.([]byte); ok {
				ret[keys[ii]] = b
			}
		}
	}
	return ret, nil
}

func (r *redisDriver) Delete(key string) error {
	conn := r.pool.Get()
	_, err := conn.Do("DEL", key)
	conn.Close()
	return err
}

func (r *redisDriver) Connection() interface{} {
	return r.pool
}

func (r *redisDriver) Close() error {
	return r.pool.Close()
}

func redisOpener(url *config.URL) (driver.Driver, error) {
	password := url.Fragment.Get("password")
	db := -1
	maxIdle := DefaultMaxIdle
	maxActive := DefaultMaxActive
	idleTimeout := DefaultIdleTimeout
	if d := url.Fragment.Get("db"); d != "" {
		val, ok := url.Fragment.Int("db")
		if !ok {
			return nil, fmt.Errorf("invalid db %q, must be an integer", d)
		}
		db = val
	}
	if v, ok := url.Fragment.Int("max_idle"); ok {
		maxIdle = v
	}
	if v, ok := url.Fragment.Int("max_active"); ok {
		maxActive = v
	}
	if v, ok := url.Fragment.Int("idle_timeout"); ok {
		idleTimeout = v
	}
	server := driver.DefaultPort(url.Value, 6379)
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			if db != -1 {
				if _, err := c.Do("SELECT", db); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		MaxIdle:     maxIdle,
		MaxActive:   maxActive,
		IdleTimeout: time.Duration(idleTimeout) * time.Second,
	}
	return &redisDriver{pool: pool}, nil
}

func init() {
	driver.Register("redis", redisOpener)
}
