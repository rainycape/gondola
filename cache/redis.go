package cache

import (
	"github.com/vmihailenco/redis"
	"net/url"
)

type RedisBackend struct {
	*redis.Client
}

func (r *RedisBackend) Set(key string, b []byte, timeout int) error {
	var req *redis.StatusReq
	if timeout == 0 {
		req = r.Client.Set(key, string(b))
	} else {
		req = r.Client.SetEx(key, int64(timeout), string(b))
	}
	return req.Err()
}

func (r *RedisBackend) Get(key string) ([]byte, error) {
	req := r.Client.Get(key)
	if err := req.Err(); err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return []byte(req.Val()), nil
}

func (r *RedisBackend) GetMulti(keys []string) (map[string][]byte, error) {
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

func (r *RedisBackend) Delete(key string) error {
	req := r.Client.Del(key)
	return req.Err()
}

func (r *RedisBackend) Connection() interface{} {
	return r.Client
}

func init() {
	RegisterBackend("redis", func(cacheUrl *url.URL) Backend {
		client := redis.NewTCPClient(cacheUrl.Host, "", -1)
		return &RedisBackend{Client: client}
	})
}
