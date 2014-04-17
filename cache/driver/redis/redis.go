// Package redis implements a Gondola cache backend using redis.
package redis

import (
	"fmt"
	"github.com/vmihailenco/redis"
	"gnd.la/cache/driver"
	"gnd.la/config"
	"strconv"
)

type redisDriver struct {
	*redis.Client
}

func (r *redisDriver) Set(key string, b []byte, timeout int) error {
	var req *redis.StatusReq
	if timeout == 0 {
		req = r.Client.Set(key, string(b))
	} else {
		req = r.Client.SetEx(key, int64(timeout), string(b))
	}
	return req.Err()
}

func (r *redisDriver) Get(key string) ([]byte, error) {
	req := r.Client.Get(key)
	if err := req.Err(); err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return []byte(req.Val()), nil
}

func (r *redisDriver) GetMulti(keys []string) (map[string][]byte, error) {
	req := r.Client.MGet(keys...)
	if err := req.Err(); err != nil {
		return nil, err
	}
	value := make(map[string][]byte, len(keys))
	for ii, v := range req.Val() {
		if v != nil {
			s := v.(string)
			value[keys[ii]] = []byte(s)
		}
	}
	return value, nil
}

func (r *redisDriver) Delete(key string) error {
	req := r.Client.Del(key)
	return req.Err()
}

func (r *redisDriver) Connection() interface{} {
	return r.Client
}

func redisOpener(url *config.URL) (driver.Driver, error) {
	password := url.Fragment.Get("password")
	db := int64(-1)
	if d := url.Fragment.Get("db"); d != "" {
		val, err := strconv.ParseInt(d, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid db %q, must be an integer", d)
		}
		db = val
	}
	conn := driver.DefaultPort(url.Value, 6379)
	client := redis.NewTCPClient(conn, password, db)
	return &redisDriver{Client: client}, nil
}

func init() {
	driver.Register("redis", redisOpener)
}
