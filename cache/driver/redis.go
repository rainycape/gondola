package driver

import (
	"github.com/vmihailenco/redis"
	"gondola/log"
	"strconv"
)

type RedisDriver struct {
	*redis.Client
}

func (r *RedisDriver) Set(key string, b []byte, timeout int) error {
	var req *redis.StatusReq
	if timeout == 0 {
		req = r.Client.Set(key, string(b))
	} else {
		req = r.Client.SetEx(key, int64(timeout), string(b))
	}
	return req.Err()
}

func (r *RedisDriver) Get(key string) ([]byte, error) {
	req := r.Client.Get(key)
	if err := req.Err(); err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return []byte(req.Val()), nil
}

func (r *RedisDriver) GetMulti(keys []string) (map[string][]byte, error) {
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

func (r *RedisDriver) Delete(key string) error {
	req := r.Client.Del(key)
	return req.Err()
}

func (r *RedisDriver) Connection() interface{} {
	return r.Client
}

func init() {
	Register("redis", func(value string, o Options) Driver {
		password := o.Get("password")
		db := int64(-1)
		if d := o.Get("db"); d != "" {
			val, err := strconv.ParseInt(d, 0, 64)
			if err == nil {
				db = val
			} else {
				log.Warningf("Invalid db %q, using default (%v)", d, db)
			}
		}
		conn := DefaultPort(value, 6379)
		client := redis.NewTCPClient(conn, password, db)
		return &RedisDriver{Client: client}
	})
}
