package cache

import (
	"encoding/gob"
	"fmt"
	_ "gondola/cache/driver/memcache"
	_ "gondola/cache/driver/redis"
	"gondola/log"
	"net"
	"reflect"
	"testing"
	"time"
)

type T interface {
	Error(...interface{})
	Errorf(string, ...interface{})
}

var (
	tests = []func(T, *Cache){
		testSetGet,
		testSetExpires,
		testDelete,
		testBytes,
	}
	benchmarks = []func(T, *Cache){
		testSetGet,
	}
)

func testPort(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

type simple struct {
	A int
	B int
	C int
}

func testSetGet(t T, c *Cache) {
	// Set object
	s1 := simple{1, 2, 3}
	if err := c.Set("s", s1, 0); err != nil {
		t.Error(err)
	}
	// Set pointer
	if err := c.Set("sp", &s1, 0); err != nil {
		t.Error(err)
	}
	// Get object from object
	var s2 simple
	err := c.Get("s", &s2)
	if err != nil {
		t.Error(err)
	} else {
		if !reflect.DeepEqual(s1, s2) {
			t.Errorf("different items (object from object): %v and %v", s1, s2)
		}
	}
	// Get object from pointer
	var s3 simple
	err = c.Get("sp", &s3)
	if err != nil {
		t.Error(err)
	} else {
		if !reflect.DeepEqual(s1, s3) {
			t.Errorf("different items (object from pointer): %v and %v", s1, s3)
		}
	}
	// Get pointer from object
	var s4 *simple
	err = c.Get("s", &s4)
	if err != nil {
		t.Error(err)
	} else {
		if !reflect.DeepEqual(s1, *s4) {
			t.Errorf("different items (pointer from object): %v and %v", s1, *s4)
		}
	}
	// Get pointer from pointer
	var s5 *simple
	err = c.Get("sp", &s5)
	if err != nil {
		t.Error(err)
	} else {
		if !reflect.DeepEqual(s1, *s5) {
			t.Errorf("different items (pointer from pointer): %v and %v", s1, *s5)
		}
	}
}

func testSetExpires(t T, c *Cache) {
	s := simple{1, 2, 3}
	if err := c.Set("s", s, 1); err != nil {
		t.Error(err)
	}
	time.Sleep(2 * time.Second)
	err := c.Get("s", nil)
	if err != ErrNotFound {
		t.Errorf("expecting ErrNotFound, got %v", err)
	}
}

func testDelete(t T, c *Cache) {
	s := simple{1, 2, 3}
	if err := c.Set("s", s, 0); err != nil {
		t.Error(err)
	}
	if err := c.Delete("s"); err != nil {
		t.Error(err)
	}
	err := c.Get("s", nil)
	if err != ErrNotFound {
		t.Errorf("expecting ErrNotFound, got %v", err)
	}
}

func testBytes(t T, c *Cache) {
	b := make([]byte, 1000)
	for ii := range b {
		b[ii] = byte(ii)
	}
	if err := c.SetBytes("b", b, 0); err != nil {
		t.Error(err)
	}
	cb, err := c.GetBytes("b")
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(b, cb) {
		t.Error("cached bytes differ")
	}
}

func testCache(t *testing.T, config string) {
	log.SetLevel(log.LDebug)
	c, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range tests {
		v(t, c)
	}
}

func TestGobCodec(t *testing.T) {
	testCache(t, "memory://?codec=gob")
}

func TestJsonCodec(t *testing.T) {
	testCache(t, "memory://?codec=json")
}

func TestCompress(t *testing.T) {
	testCache(t, "memory://?min_compress=0&compress_level=9")
}

func TestPrefix(t *testing.T) {
	prefix := "foo"
	c1, err := New("memory://?prefix=" + prefix)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := New("memory://")
	if err != nil {
		t.Fatal(err)
	}
	s1 := simple{1, 2, 3}
	key := "spre"
	if err := c1.Set(key, s1, 0); err != nil {
		t.Fatal(err)
	}
	if err := c2.Get(key, nil); err != ErrNotFound {
		t.Errorf("expecting ErrNotFound, got %v", err)
	}
	var s2 simple
	err = c2.Get(prefix+key, &s2)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(s1, s2) {
		t.Errorf("different objects %v and %v", s1, s2)
	}
}

func TestMemcache(t *testing.T) {
	if !testPort(11211) {
		t.Skip("memcache is not running. start memcache on localhost to run this test")
	}
	testCache(t, "memcache://127.0.0.1")
}

func TestRedis(t *testing.T) {
	if !testPort(6379) {
		t.Skip("redis is not running. start redis on localhost to run this test")
	}
	testCache(t, "redis://127.0.0.1")
}

func benchmarkCache(b *testing.B, config string) {
	c, err := New(config)
	if err != nil {
		b.Fatal(err)
	}
	log.SetLevel(log.LInfo)
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		for _, v := range benchmarks {
			v(b, c)
		}
	}
}

func BenchmarkGobCache(b *testing.B) {
	benchmarkCache(b, "memory://?codec=gob")
}

func BenchmarkJsonCache(b *testing.B) {
	benchmarkCache(b, "memory://?codec=json")
}

func BenchmarkPrefixCache(b *testing.B) {
	benchmarkCache(b, "memory://?prefix=foo")
}

func BenchmarkPrefixJsonCache(b *testing.B) {
	benchmarkCache(b, "memory://?prefix=foo&codec=gob")
}

func init() {
	gob.Register((*simple)(nil))
}
