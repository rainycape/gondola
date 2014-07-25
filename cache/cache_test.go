package cache

import (
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	_ "gnd.la/cache/driver/memcache"
	_ "gnd.la/cache/driver/redis"
	"gnd.la/config"
	_ "gnd.la/encoding/codec/msgpack"
	"gnd.la/log"
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
		testSetGetMultiHomogeneous,
		testSetGetMultiHeterogeneous,
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

func testSetGetMultiHomogeneous(t T, c *Cache) {
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
	out1 := make(map[string]interface{}, len(items))
	for _, k := range keys {
		out1[k] = nil
	}
	err := c.GetMulti(out1, UniTyper([]int(nil)))
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("result from multi-get %v", out1)
		if len(out1) != len(items) {
			t.Errorf("expecting %d items in multi get, got %d", len(items), len(out1))
		}
		for ii, v := range items {
			key := keys[ii]
			if !deepEqual(v, out1[key]) {
				t.Errorf("object in multi get different, %v (%T) and %v (%T)", v, v, out1[key], out1[key])
			}
		}
	}
	out2 := make(map[string]interface{}, len(items))
	for _, k := range keys {
		out2[k] = []int(nil)
	}
	err = c.GetMulti(out2, nil)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("result from multi-get %v", out2)
		if len(out2) != len(items) {
			t.Errorf("expecting %d items in multi get, got %d", len(items), len(out2))
		}
		for ii, v := range items {
			key := keys[ii]
			if !deepEqual(v, out1[key]) {
				t.Errorf("object in multi get different, %v (%T) and %v (%T)", v, v, out2[key], out2[key])
			}
		}
	}
}

func testSetGetMultiHeterogeneous(t T, c *Cache) {
	k1 := "k1"
	v1 := 42
	k2 := "k2"
	v2 := []float64{1, 2, 3, 4}
	k3 := "k3"
	v3 := "foobar"
	if err := c.Set(k1, v1, 0); err != nil {
		t.Error(err)
	}
	if err := c.Set(k2, v2, 0); err != nil {
		t.Error(err)
	}
	if err := c.Set(k3, v3, 0); err != nil {
		t.Error(err)
	}
	out := map[string]interface{}{
		"k1": int(0),
		"k2": []float64(nil),
		"k3": "",
		"k4": nil, // this one should be deleted
	}
	if err := c.GetMulti(out, nil); err != nil {
		t.Error(err)
	}
	t.Logf("heterogeneous multi-get: %v", out)
	if len(out) != 3 {
		t.Errorf("expecting 3 items in heterogeneous multi-get, got %d", len(out))
	}
	if !reflect.DeepEqual(out[k1], v1) {
		t.Errorf("bad value for k1 - want %v, got %v", v1, out[k1])
	}
	if !reflect.DeepEqual(out[k2], v2) {
		t.Errorf("bad value for k2 - want %v, got %v", v2, out[k2])
	}
	if !reflect.DeepEqual(out[k3], v3) {
		t.Errorf("bad value for k3 - want %v, got %v", v3, out[k3])
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
	testCache(t, "memory://#codec=gob")
}

func TestJsonCodec(t *testing.T) {
	testCache(t, "memory://#codec=json")
}

func TestMsgpackCodec(t *testing.T) {
	testCache(t, "memory://#codec=msgpack")
}

func TestCompress(t *testing.T) {
	testCache(t, "memory://#min_compress=0&compress_level=9")
}

func TestPrefix(t *testing.T) {
	prefix := "foo"
	c1, err := newCache("memory://#prefix=" + prefix)
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

func TestMemoryCacheMaxSize(t *testing.T) {
	c, err := newCache("memory://#max_size=1K")
	if err != nil {
		t.Fatal(err)
	}
	data1 := make([]byte, 256)
	data2 := make([]byte, 512)
	c.SetBytes("k1", data1, 0)
	c.SetBytes("k2", data2, 0)
	c.SetBytes("k3", data2, 0)
	// Sleep for 0.1s so the goroutine has time
	// to remove items.
	time.Sleep(100 * time.Millisecond)
	// Should have evicted k2 or k3
	k1d, err := c.GetBytes("k1")
	if err != nil {
		t.Error(err)
	}
	if len(k1d) != len(data1) {
		t.Errorf("bad data for key k1")
	}
	_, err2 := c.GetBytes("k2")
	_, err3 := c.GetBytes("k3")
	if err2 == nil && err3 == nil {
		t.Errorf("should have evicted k2 or k3")
	}
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
	benchmarkCache(b, "memory://#codec=gob")
}

func BenchmarkJsonCache(b *testing.B) {
	benchmarkCache(b, "memory://#codec=json")
}

func BenchmarkMemcacheCache(b *testing.B) {
	if !testPort(11211) {
		b.Skip("memcache is not running. start memcache on localhost to run this test")
	}
	benchmarkCache(b, "memcache://#codec=json")
}

func BenchmarkRedisCache(b *testing.B) {
	if !testPort(6379) {
		b.Skip("redis is not running. start redis on localhost to run this test")
	}
	benchmarkCache(b, "redis://#codec=json")
}

func BenchmarkMemcacheMsgpackCache(b *testing.B) {
	if !testPort(11211) {
		b.Skip("memcache is not running. start memcache on localhost to run this test")
	}
	benchmarkCache(b, "memcache://#codec=msgpack")
}

func BenchmarkMsgpackCache(b *testing.B) {
	benchmarkCache(b, "memory://#codec=msgpack")
}

func BenchmarkPrefixCache(b *testing.B) {
	benchmarkCache(b, "memory://#prefix=foo&codec=gob")
}

func init() {
	gob.Register((*simple)(nil))
}
