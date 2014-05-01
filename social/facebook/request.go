package facebook

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gnd.la/net/httpclient"
)

func graphUrl(path string, secure bool) string {
	var proto string
	if secure {
		proto = "https"
	} else {
		proto = "http"
	}
	separator := "/"
	if strings.HasPrefix(path, "/") {
		separator = ""
	}
	return fmt.Sprintf("%v://graph.facebook.com%v%v", proto, separator, path)
}

func graphRead(resp *httpclient.Response, err error) (map[string]interface{}, error) {
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if responseHasError(resp) {
		return nil, decodeResponseError(resp)
	}
	var m map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (app *App) Get(path string, parameters map[string]string, accessToken string) (map[string]interface{}, error) {
	secure := false
	v := url.Values{}
	for k1, v1 := range parameters {
		v.Add(k1, v1)
	}
	if accessToken != "" {
		secure = true
		v.Add("access_token", accessToken)
	}
	requestUrl := graphUrl(path, secure)
	if len(v) > 0 {
		requestUrl += fmt.Sprintf("?%v", v.Encode())
	}
	resp, err := app.client().Get(requestUrl)
	return graphRead(resp, err)
}

func (app *App) Post(path string, parameters map[string]string, accessToken string) (map[string]interface{}, error) {
	secure := false
	v := url.Values{}
	for k1, v1 := range parameters {
		v.Add(k1, v1)
	}
	if accessToken != "" {
		secure = true
		v.Add("access_token", accessToken)
	}
	requestUrl := graphUrl(path, secure)
	resp, err := app.client().PostForm(requestUrl, v)
	return graphRead(resp, err)
}
