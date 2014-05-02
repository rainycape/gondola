package httpclient_test

import (
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"gnd.la/net/httpclient"
	"gnd.la/util/urlutil"

	"github.com/elazarl/goproxy"
)

const (
	httpBin = "http://httpbin.org"
)

func testUserAgent(t *testing.T, c *httpclient.Client, exp string) {
	ep := urlutil.MustJoin(httpBin, "/user-agent")
	resp, err := c.Get(ep)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()
	var m map[string]interface{}
	if err := resp.JSONDecode(&m); err != nil {
		t.Fatal(err)
	}
	ua := m["user-agent"].(string)
	if idx := strings.Index(ua, " AppEngine-Google"); idx >= 0 {
		ua = ua[:idx]
	}
	if ua != exp {
		t.Errorf("expecting User-Agent %q, got %q instead", exp, ua)
	}
}

func TestUserAgent(t *testing.T) {
	const ua = "Gondolier"
	c := httpclient.New(nil)
	testUserAgent(t, c, httpclient.DefaultUserAgent)
	c.SetUserAgent(ua)
	testUserAgent(t, c, ua)
	testUserAgent(t, c.Clone(nil), ua)
}

func decodeArgs(resp *httpclient.Response) (map[string]string, error) {
	var m map[string]interface{}
	if err := resp.JSONDecode(&m); err != nil {
		return nil, err
	}
	var args map[string]interface{}
	if strings.HasSuffix(resp.Request.URL.Path, "post") {
		args = m["form"].(map[string]interface{})
	} else {
		args = m["args"].(map[string]interface{})
	}
	values := make(map[string]string, len(args))
	for k, v := range args {
		values[k] = v.(string)
	}
	return values, nil
}

func testForm(t *testing.T, f func(string, url.Values) (*httpclient.Response, error), u string, data map[string]string, exp map[string]string) {
	form := make(url.Values)
	for k, v := range data {
		form.Add(k, v)
	}
	resp, err := f(u, form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()
	args, err := decodeArgs(resp)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, exp) {
		t.Errorf("expecting values %v, got %v instead", exp, args)
	}
}

func TestGetForm(t *testing.T) {
	data := map[string]string{"a": "b", "c": "d"}
	f := httpclient.New(nil).GetForm
	testForm(t, f, urlutil.MustJoin(httpBin, "/get"), data, data)
	expect := map[string]string{"e": "f"}
	for k, v := range data {
		expect[k] = v
	}
	testForm(t, f, urlutil.MustJoin(httpBin, "/get?e=f"), data, expect)
}

func TestPostForm(t *testing.T) {
	data := map[string]string{"a": "b", "c": "d"}
	testForm(t, httpclient.New(nil).PostForm, urlutil.MustJoin(httpBin, "/post"), data, data)
}

func redirNumber(t *testing.T, url string) int {
	b := path.Base(url)
	val, err := strconv.Atoi(b)
	if err != nil {
		t.Fatal(err)
	}
	return val
}

func TestRedirect(t *testing.T) {
	start := urlutil.MustJoin(httpBin, "/redirect/6")
	end := urlutil.MustJoin(httpBin, "/get")
	c := httpclient.New(nil)
	resp, err := c.Get(start)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()
	if u := resp.Request.URL.String(); u != end {
		t.Errorf("expecting final url %q, got %q instead", end, u)
	}
	cur := redirNumber(t, start)
	next := start
	for {
		req, err := http.NewRequest("GET", next, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := c.Trip(req)
		if err != nil {
			t.Fatal(err)
		}
		if cur > 0 {
			redir, err := resp.Redirect()
			if err != nil {
				t.Fatal(err)
			}
			cur--
			if cur > 0 {
				rn := redirNumber(t, redir)
				if rn != cur {
					t.Fatalf("expecting redirect %d, got %d instead", cur, rn)
				}
			}
			next = redir
		} else {
			if resp.IsRedirect() {
				t.Error("unexpected redirect")
			}
			if u := resp.Request.URL.String(); u != end {
				t.Errorf("expecting final url %q, got %q instead", end, u)
			}
			break
		}
	}
}

func TestProxy(t *testing.T) {
	c := httpclient.New(nil)
	if !c.SupportsProxy() {
		t.Skipf("current run environment does not support support proxies")
	}
	const addr = "127.0.0.1:12345"
	proxy := goproxy.NewProxyHttpServer()
	count := 0
	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			count++
			return r, nil
		})
	go func() {
		http.ListenAndServe(addr, proxy)
	}()
	time.Sleep(time.Millisecond)
	c.SetProxy(func(_ *http.Request) (*url.URL, error) {
		return url.Parse("http://" + addr)
	})
	testUserAgent(t, c, httpclient.DefaultUserAgent)
	if count != 1 {
		t.Errorf("expecting 1 proxy request, got %d instead", count)
	}
	testUserAgent(t, c.Clone(nil), httpclient.DefaultUserAgent)
	if count != 2 {
		t.Errorf("expecting 2 proxy request, got %d instead", count)
	}
}
