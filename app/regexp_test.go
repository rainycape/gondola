package app

import (
	"regexp"
	"testing"
)

type regexpTest struct {
	pattern string
	args    []interface{}
	expect  string
}

type regexpFailureTest struct {
	pattern string
	args    []interface{}
}

var (
	regexpTests = []regexpTest{
		{"^/program/(\\d+)/$", []interface{}{1}, "/program/1/"},
		{"^/program/(\\d+)/$", []interface{}{12345}, "/program/12345/"},

		{"^/program/(\\d+)/version/(\\d+)/$", []interface{}{1, 2}, "/program/1/version/2/"},
		{"^/program/(?P<pid>\\d+)/version/(?P<vers>\\d+)/$", []interface{}{1, 2}, "/program/1/version/2/"},

		{"^/program/(\\d+)/(?:version/(\\d+)/)?$", []interface{}{1}, "/program/1/"},
		{"^/program/(\\d+)/(?:version/(\\d+)/)?$", []interface{}{1, 2}, "/program/1/version/2/"},

		{"^/program/(\\d+)/(?:version/(\\d+)/)?(?:revision/(\\d+)/)?$", []interface{}{1}, "/program/1/"},
		{"^/program/(\\d+)/(?:version/(\\d+)/)?(?:revision/(\\d+)/)?$", []interface{}{1, 2}, "/program/1/version/2/"},
		{"^/program/(\\d+)/(?:version/(\\d+)/)?(?:revision/(\\d+)/)?$", []interface{}{1, 2, 3}, "/program/1/version/2/revision/3/"},

		{"^/archive/(?:(\\d{4})(\\d{2})(\\d{2})/)?$", nil, "/archive/"},
		{"^/archive/(?:(\\d{4})(\\d{2})(\\d{2})/)?$", []interface{}{1970, "01", "01"}, "/archive/19700101/"},

		{"^/image/(\\w+)\\.(\\w+)$", []interface{}{"test", "png"}, "/image/test.png"},
		{"^/image/(\\w+)\\-(\\w+)$", []interface{}{"test", "png"}, "/image/test-png"},
		{"^/image/(\\w+)\\\\(\\w+)$", []interface{}{"test", "png"}, "/image/test\\png"},

		{"^/section/(?:sub/(\\d+))?(?:/subsub/(\\d+))?$", nil, "/section/"},
		{"^/section/(?:sub/(\\d+))?(?:/subsub/(\\d+))?$", []interface{}{1}, "/section/sub/1"},
		{"^/section/(?:sub/(\\d+))?(?:/subsub/(\\d+))?$", []interface{}{1, 2}, "/section/sub/1/subsub/2"},

		{`^/github\.com/(?P<github_repo>[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+)\.v(?P<version>\d+)(?P<sub>(/\w+)*)?$`, []interface{}{"foo/bar", 1}, "/github.com/foo/bar.v1"},
	}

	regexpFailureTests = []regexpFailureTest{
		{"^/program/(\\d+)/$", nil},
		{"^/program/(\\d+)/$", []interface{}{"foo"}},
		{"^/program/(\\d+)/$", []interface{}{1, 2}},

		{"^/program/(\\d+)/(?:version/(\\d+)/)?(?:revision/(\\d+)/)?$", []interface{}{1, 2, 3, 4}},
	}
)

func testRegexp(t *testing.T, pattern string, args []interface{}, expected string) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		t.Error(err)
		return
	}
	res, err := formatRegexp(r, args)
	if err != nil {
		t.Error(err)
		return
	}
	if res != expected {
		t.Errorf("expecting format(%q) with args %v = %q, got %q instead", pattern, args, expected, res)
	}

}

func TestRegexp(t *testing.T) {
	for _, v := range regexpTests {
		testRegexp(t, v.pattern, v.args, v.expect)
	}
}

func TestBadRegexp(t *testing.T) {
	for _, v := range regexpFailureTests {
		r, err := regexp.Compile(v.pattern)
		if err != nil {
			t.Error(err)
			continue
		}
		res, err := formatRegexp(r, v.args)
		if err == nil {
			t.Errorf("expecting an error formatting %q with args %v = %q, got %q instead", v.pattern, v.args, res)
		}
	}
}
