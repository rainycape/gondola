package assets

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Service indicates the base URL for the assets
// service to use. POST calls will be made to:
//
//  Reducer + "css"
//  Reducer + "js"
//  Reducer + "less"
//  Reducer + "coffee"
//  ...
//
// The code to reduce or compile will be sent in
// the form parameter named "code".
var Service = "http://assets.gondolaweb.com/"

func assetsService(path string, w io.Writer, r io.Reader) (int, int, error) {
	code, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, 0, err
	}
	form := url.Values{
		"code": []string{string(code)},
	}
	resp, err := http.PostForm(Service+path, form)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := ioutil.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("invalid %s code: %s", path, string(msg))
	}
	n, err := io.Copy(w, resp.Body)
	return len(code), int(n), err
}
