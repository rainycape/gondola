package assets

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

func reducer(path string, w io.Writer, r io.Reader) (int, int, error) {
	code, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, 0, err
	}
	form := url.Values{
		"file": []string{string(code)},
	}
	resp, err := http.PostForm("http://gondola-reducer.appspot.com/"+path, form)
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
