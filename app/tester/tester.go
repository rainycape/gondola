// Package tester implements functions for testing and benchmarking
// Gondola applications.
//
// To set up a test, create a _test.go file, like in Go standard tests,
// and create a standard Go test function. Then, use New to create a
// Tester and use it to generate requests and check what they return.
//
// See Tester or this package's tests for a few examples of complete tests.
// For benchmark, use Request.Bench. See its documentation for details.
//
// Additionaly, tests might be run against a remote server by using the
// -H command line flag. e.g.
//
//  go test -v -H example.com // or https://example.com
//
// Will run your tests against the server at example.com, rather than
// using the compiled code. This is pretty useful to make sure your
// tests pass on production after deploying your application.
//
// App Engine apps with a correctly set up app.yaml might also use the -R
// flag to automatically test against http://<your-app-id>.appspot.com.
//
//  goapp test -v -R
package tester

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"gnd.la/app"
	"gnd.la/internal"
	"gnd.la/util/types"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	remoteHost *string
	gaeRemote  *bool
)

// Reporter is the interface used to log and
// report errors when testing Gondola applications.
// Both testing.T and testing.B implement this
// interface.
type Reporter interface {
	Log(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
}

// Contains is an alias for string, to indicate Request.Expect
// to check that a response contains the given value (rather
// than being equal). Users should use the shorthand methods
// Response.Contains and Response.ContainsHeader.
type Contains string

// Match is an alias for string, to indicate Request.Expect to
// compile the given value as a regular expression and then
// match it against the response body. Users should use the
// shorthand methods Response.Match and Response.MatchHeader.
type Match string

type readCloser struct {
	io.Reader
}

func (r readCloser) Close() error {
	return nil
}

type response struct {
	headerWritten bool
	code          int
	body          bytes.Buffer
	header        http.Header
	stacks        [][]byte
	err           error
	bench         bool
}

func newRemoteResponse(r *http.Response, err error) *response {
	var code int
	var body []byte
	var header http.Header
	if err == nil {
		defer r.Body.Close()
		code = r.StatusCode
		header = r.Header
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			err = fmt.Errorf("error reading body: %s", err)
		}
	}
	return &response{
		code:   code,
		body:   *bytes.NewBuffer(body),
		header: header,
		err:    err,
	}
}

func (r *response) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

func (r *response) Write(b []byte) (int, error) {
	if !r.headerWritten {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(b)
}

func (r *response) WriteHeader(code int) {
	stack := make([]byte, 8192)
	stack = stack[:runtime.Stack(stack, false)]
	r.stacks = append(r.stacks, stack)
	if r.headerWritten {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "WriteHeader() called %d times\n", len(r.stacks))
		for ii, v := range r.stacks {
			fmt.Fprintf(&buf, "Call %d\n", ii+1)
			buf.WriteString(string(v))
			buf.WriteString("\n")
		}
		r.err = errors.New(buf.String())
		return
	}
	r.headerWritten = true
	if code == 0 {
		r.err = fmt.Errorf("WriteHeader() called with invalid code 0\nStack:\n%s", string(r.stacks[len(r.stacks)-1]))
	}
	r.code = code
}

// benchResponse is used for benchmarks. It does nothing
// on WriteHeader() and Write(), since this removes the
// noise caused by copying the response bytes to the
// buffer
type benchResponse struct {
	header http.Header
}

func (r *benchResponse) Header() http.Header         { return r.header }
func (r *benchResponse) Write(b []byte) (int, error) { return len(b), nil }
func (r *benchResponse) WriteHeader(code int)        {}

// A Request represents a request to be sent to
// the app. Users won't usually need to construct
// requests by hand, most of the time the helper
// type Test and its conveniency methods should
// be used.
// Use Request.Expect and its related functions
// to perform tests on the App response.
type Request struct {
	Reporter Reporter
	App      *app.App
	Header   http.Header
	Method   string
	Path     string
	Body     []byte
	err      error
	resp     *response
}

func (r *Request) asHTTPRequest() (*http.Request, error) {
	var u *url.URL
	var err error
	var host string
	var requestURI string
	if *remoteHost != "" {
		base := *remoteHost
		if !strings.Contains(base, "://") {
			base = "http://" + base
		}
		if base[len(base)-1] == '/' {
			base = base[:len(base)-1]
		}
		u, err = url.Parse(base + r.Path)
		if u != nil {
			r.Header.Add("Host", u.Host)
			host = u.Host
		}
	} else {
		host = r.Header.Get("Host")
		if host == "" {
			host = "localhost"
		}
		u, err = url.Parse(r.Path)
		requestURI = r.Path
	}
	if err != nil {
		return nil, err
	}
	return &http.Request{
		Method:        r.Method,
		URL:           u,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        r.Header,
		Body:          readCloser{bytes.NewReader(r.Body)},
		ContentLength: int64(len(r.Body)),
		Host:          host,
		RequestURI:    requestURI,
	}, nil
}

func (r *Request) setErr(err error) {
	if err != nil {
		r.err = err
		r.Reporter.Error(r.err)
	}
}

func (r *Request) errorf(format string, args ...interface{}) {
	r.setErr(fmt.Errorf(format, args...))
}

func (r *Request) do() bool {
	if r.resp == nil {
		if r.err == nil {
			req, err := r.asHTTPRequest()
			if err != nil {
				r.setErr(err)
			} else {
				r.Reporter.Log(fmt.Sprintf("requesting %s", req.URL))
				start := time.Now()
				if *remoteHost != "" {
					client := &http.Client{}
					resp, err := client.Do(req)
					r.resp = newRemoteResponse(resp, err)
				} else {
					r.resp = new(response)
					r.App.ServeHTTP(r.resp, req)
				}
				r.Reporter.Log(fmt.Sprintf("received response (%d bytes) with code %d in %s", r.resp.body.Len(), r.resp.code, time.Since(start)))
				r.setErr(r.resp.err)
			}
		}
	}
	return r.err == nil
}

// Bench performs a benchmark with this request. Note that Bench
// does not perform any tests on the response, so you should not
// use Bench for testing, only strictly for benchmarking.
// Tipically, Bench is used from a testing benchmark.
//
//  BenchmarkSomething(b *testing.B) {
//	te := tester.New(b, MyApp)
//	te.Get("/something", nil).Bench(b)
//  }
//
// Note that bench does not support benchmarking a remote server,
// so it ignores the -H flag.
func (r *Request) Bench(b *testing.B) {
	req, err := r.asHTTPRequest()
	if err != nil {
		r.Reporter.Fatal(err)
		return
	}
	resp := new(benchResponse)
	resp.header = make(http.Header)
	// do the boxing once, rather than on
	// on every iteration
	var w http.ResponseWriter = resp
	r.App.Debug = false
	r.App.TemplateDebug = false
	// Do the request once, to allow templates
	// to be cached
	r.App.ServeHTTP(w, req)
	b.ReportAllocs()
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		r.App.ServeHTTP(w, req)
	}
}

// Err returns the first error generated from this Request.
func (r *Request) Err() error {
	return r.err
}

// AddHeader is a conveniency function which adds a new HTTP header
// and returns the same *Request, to allow chaining. The value is
// converted to an string using types.ToString.
func (r *Request) AddHeader(key string, value interface{}) *Request {
	if r.resp != nil {
		panic("can't add header after sending request")
	}
	r.Header.Add(key, types.ToString(value))
	return r
}

func (r *Request) expect(what interface{}, name string, value interface{}) *Request {
	if r.do() {
		var s string
		switch v := value.(type) {
		case string:
			s = v
		case []byte:
			s = string(v)
		default:
			s = types.ToString(value)
		}
		switch x := what.(type) {
		case int:
			val, err := strconv.ParseInt(s, 0, 64)
			if err != nil {
				r.errorf("expecting %s = %d, got non numeric value %q instead", name, x, s)
			} else if val != int64(x) {
				r.errorf("expecting %s = %d, got %d instead", name, x, val)
			}
		case []byte:
			if val, ok := value.([]byte); ok {
				if !bytes.Equal(val, x) {
					r.errorf("expecting %s = %v, got %v instead", name, x, val)
				}
			} else {
				if s != string(x) {
					r.errorf("expecting %s = %q, got %q instead", name, string(x), s)
				}
			}
		case string:
			if s != x {
				r.errorf("expecting %s = %q, got %q instead", name, x, s)
			}
		case Contains:
			if !strings.Contains(s, string(x)) {
				r.errorf("expecting %s containing %q, got %q instead", name, x, s)
			}
		case Match:
			re, err := regexp.Compile(string(x))
			if err != nil {
				r.err = fmt.Errorf("error compiling regular expression %q: %s", x, err)
				r.Reporter.Fatal(r.err)
				break
			}
			if !re.MatchString(s) {
				r.errorf("expecting %s matching %q, got %q instead", name, x, s)
			}
		case io.Reader:
			data, err := ioutil.ReadAll(x)
			if err != nil {
				r.errorf("error reading expected body: %s", err)
				break
			}
			return r.expect(data, name, value)
		case nil:
			if val, ok := value.([]byte); ok {
				if len(val) > 0 {
					r.errorf("expecting empty %s, got %v instead", name, val)
				}
			} else if len(s) > 0 {
				r.errorf("expecting empty %s, got %q instead", name, s)
			}
		default:
			r.Reporter.Fatal(fmt.Errorf("don't know what to expect from %T", what))
		}
	}
	return r
}

// ExpectHeader works like Expect, but checks the requested header
// rather than the response body. See Response.Expect to find the
// accepted types.
func (r *Request) ExpectHeader(name string, what interface{}) *Request {
	value := ""
	if r.resp != nil && r.resp.header != nil {
		value = r.resp.header.Get(name)
	}
	return r.expect(what, fmt.Sprintf("header %q", name), value)
}

// MatchHeader works like Match, but checks the requested header
// rather than the response body.
func (r *Request) MatchHeader(name string, what string) *Request {
	return r.ExpectHeader(name, Match(what))
}

// ContainsHeader works like Contains, but checks the requested header
// rather than the response body.
func (r *Request) ContainsHeader(name string, what string) *Request {
	return r.ExpectHeader(name, Contains(what))
}

// Expect checks the response body or the status against the given criteria.
// Expect accepts the following argument types.
//
//  int: Check that the response code matches the given value.
//  nil: Check that the response is empty.
//  string: Check that the response body is equal to the given value.
//  Contains: Check that the response body contains the given value.
//  Match: Compile the given value as a regular expression and match it against the body.
//
// Request send the request to the App lazily, on the first call to
// Expect. Multiple Expect (and related functions, like Contains or
// ExpectHeader) can be chained, but processing will stop as soon as
// an error is generated (either when getting the response from the App
// or from a failed Expect check). To get the first error, use the
// Err() method. Note that when using a *testing.T or a *testing.B
// as the Reporter, Expect will call t.Error or t.Fatal for you.
// See also the shorthand functions Request.Contains and Request.Match.
// Additionally, check this package's tests, in tester_test.go, for
// examples.
func (r *Request) Expect(what interface{}) *Request {
	if r.do() {
		if code, ok := what.(int); ok {
			if code != r.resp.code {
				r.setErr(fmt.Errorf("expecting status code %d, got %d instead", code, r.resp.code))
			}
			return r
		}
		return r.expect(what, "body", r.resp.body.Bytes())
	}
	// if r.do() is false we might have no r.resp
	return r
}

// Contains checks that the response body contains the given string.
// It's a shorthand for r.Expect(tester.Contains(what)).
func (r *Request) Contains(what string) *Request {
	return r.Expect(Contains(what))
}

// Match checks that the response body matches the given regular
// expression. It's a shorthand for r.Expect(tester.Match(what)).
// If the regular expression can't be compiled, the
// current test will be aborted.
func (r *Request) Match(what string) *Request {
	return r.Expect(Match(what))
}

// Tester encapsulates the app.App to be tested
// with the Reporter to send the logs and errors
// to. It's usually used in conjuction with the
// testing package.
//
//  func TestMyApp(t *testing.T) {
//	// App was initialized on init. Note that
//	// the app does not need to be listening.
//	te := tester.New(t, App)
//	te.Get("/foo", nil).Expect(200)
//  }
//
type Tester struct {
	Reporter Reporter
	App      *app.App
}

// New returns prepares the *app.App and then
// returns a new Tester. It also disables
// logging in the App, since the Tester does
// its own logging.
func New(r Reporter, a *app.App) *Tester {
	if !flag.Parsed() {
		flag.Parse()
	}
	if *gaeRemote && *remoteHost == "" {
		h := internal.AppEngineAppHost()
		remoteHost = &h
	}
	if err := a.Prepare(); err != nil {
		r.Fatal(fmt.Errorf("error preparing app: %s", err))
	}
	a.Logger = nil
	return &Tester{r, a}
}

// Request returns a new request with the given method, path and body. Body
// might be one of the following:
//
//  string
//  []byte
//  io.Reader
func (t *Tester) Request(method string, path string, body interface{}) *Request {
	var data []byte
	var err error
	if body != nil {
		switch x := body.(type) {
		case nil:
		case string:
			data = []byte(x)
		case []byte:
			data = x
		case io.Reader:
			data, err = ioutil.ReadAll(x)
		default:
			err = fmt.Errorf("can't send %T as body", body)
		}
	}
	if err != nil {
		t.Reporter.Fatal(err)
	}
	return &Request{
		Reporter: t.Reporter,
		App:      t.App,
		Header:   make(http.Header),
		Method:   method,
		Path:     path,
		Body:     data,
		err:      err,
	}
}

// Get returns a GET request with the given path and parameters, which
// are appended to the URL as a query string.
func (t *Tester) Get(path string, params map[string]interface{}) *Request {
	if len(params) > 0 {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		path += sep + encode(params)
	}
	return t.Request("GET", path, nil)
}

// Post returns a POST request with the given body. See Tester.Request
// for the supported types in the body argument.
func (t *Tester) Post(path string, body interface{}) *Request {
	return t.Request("POST", path, body)
}

// Form returns a POST request which sends a form with the given parameters,
// while also setting the right Content-Type.
func (t *Tester) Form(path string, params map[string]interface{}) *Request {
	var body interface{}
	if len(params) > 0 {
		body = encode(params)
	}
	req := t.Request("POST", path, body)
	req.AddHeader("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func encode(params map[string]interface{}) string {
	values := make(url.Values)
	for k, v := range params {
		values.Add(k, types.ToString(v))
	}
	return values.Encode()
}

func init() {
	if internal.InTest() {
		remoteHost = flag.String("H", "", "Host to run the test against")
		if h := internal.AppEngineAppHost(); h != "" {
			gaeRemote = flag.Bool("R", false, fmt.Sprintf("Run tests against %s", h))
		}
	}
}
