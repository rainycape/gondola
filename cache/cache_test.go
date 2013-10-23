package cache

import (
	"encoding/gob"
	"fmt"
	_ "gnd.la/cache/codec/msgpack"
	_ "gnd.la/cache/driver/memcache"
	_ "gnd.la/cache/driver/redis"
	"gnd.la/config"
	"gnd.la/log"
	"net"
	"reflect"
	"testing"
	"time"
)

type T interface {
	Error(...interface{})
	Errorf(string, ...interface{})
	Logf(string, ...interface{})
}

var (
	tests = []func(T, *Cache){
		testSetGet,
		testSetGetBasic,
		testSetGetMulti,
		testSetExpires,
		testDelete,
		testBytes,
	}
	benchmarks = []func(T, *Cache){
		testSetGet,
	}
)

func newCache(url string) (*Cache, error) {
	return New(config.MustParseURL(url))
}

func testPort(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func deepEqual(a interface{}, b interface{}) bool {
	return reflect.DeepEqual(a, b)
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
		if !deepEqual(s1, s2) {
			t.Errorf("different items (object from object): %v and %v", s1, s2)
		}
	}
	// Get object from pointer
	var s3 simple
	err = c.Get("sp", &s3)
	if err != nil {
		t.Error(err)
	} else {
		if !deepEqual(s1, s3) {
			t.Errorf("different items (object from pointer): %v and %v", s1, s3)
		}
	}
	// Get pointer from object
	var s4 *simple
	err = c.Get("s", &s4)
	if err != nil {
		t.Error(err)
	} else {
		if !deepEqual(s1, *s4) {
			t.Errorf("different items (pointer from object): %v and %v", s1, *s4)
		}
	}
	// Get pointer from pointer
	var s5 *simple
	err = c.Get("sp", &s5)
	if err != nil {
		t.Error(err)
	} else {
		if !deepEqual(s1, *s5) {
			t.Errorf("different items (pointer from pointer): %v and %v", s1, *s5)
		}
	}
}

func testSetGetBasic(t T, c *Cache) {
	a1 := 42
	if err := c.Set("a", a1, 0); err != nil {
		t.Error(err)
	}
	var a2 int
	if err := c.Get("a", &a2); err != nil {
		t.Error(err)
	}
	if !deepEqual(a1, a2) {
		t.Errorf("%v != %v", a1, a2)
	}
	s1 := "fortytwo"
	if err := c.Set("s", s1, 0); err != nil {
		t.Error(err)
	}
	var s2 string
	if err := c.Get("s", &s2); err != nil {
		t.Error(err)
	}
	if !deepEqual(s1, s2) {
		t.Errorf("%q != %q", s1, s2)
	}
}

func testSetGetMulti(t T, c *Cache) {
	items := [][]int{
		[]int{0},
		[]int{1},
		[]int{2},
		[]int{3},
		[]int{4},
	}
	keys := make([]string, len(items))
	for ii, v := range items {
		key := fmt.Sprintf("i:%d", ii)
		if err := c.Set(key, v, 0); err != nil {
			t.Error(err)
		}
		keys[ii] = key
	}
	objs, err := c.GetMulti(keys, (*[]int)(nil))
	//objs, err := c.GetMulti(keys, nil)
	t.Logf("result from multi-get %v", objs)
	if err != nil {
		t.Error(err)
	}
	if len(objs) != len(items) {
		t.Errorf("expecting %d items in multi get, got %d", len(items), len(objs))
	}
	for ii, v := range items {
		key := fmt.Sprintf("i:%d", ii)
		if !deepEqual(v, objs[key]) {
			t.Errorf("object in multi get different, %v (%T) and %v (%T)", v, v, objs[key], objs[key])
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
	} else if !deepEqual(b, cb) {
		t.Error("cached bytes differ")
	}
}

func testCache(t *testing.T, url string) {
	if testing.Verbose() {
		log.SetLevel(log.LDebug)
	}
	c, err := newCache(url)
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

func TestMsgpackCodec(t *testing.T) {
	testCache(t, "memory://?codec=msgpack")
}

func TestCompress(t *testing.T) {
	testCache(t, "memory://?min_compress=0&compress_level=9")
}

func TestPrefix(t *testing.T) {
	prefix := "foo"
	c1, err := newCache("memory://?prefix=" + prefix)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := newCache("memory://")
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
	} else if !deepEqual(s1, s2) {
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
	c, err := newCache(config)
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

func BenchmarkMemcacheCache(b *testing.B) {
	if !testPort(11211) {
		b.Skip("memcache is not running. start memcache on localhost to run this test")
	}
	benchmarkCache(b, "memcache://?codec=json")
}

func BenchmarkRedisCache(b *testing.B) {
	if !testPort(6379) {
		b.Skip("redis is not running. start redis on localhost to run this test")
	}
	benchmarkCache(b, "redis://?codec=json")
}

func BenchmarkMsgpackCache(b *testing.B) {
	benchmarkCache(b, "memory://?codec=msgpack")
}

func BenchmarkPrefixCache(b *testing.B) {
	benchmarkCache(b, "memory://?prefix=foo&codec=gob")
}

func init() {
	gob.Register((*simple)(nil))
}
