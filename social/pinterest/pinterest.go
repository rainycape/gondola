package pinterest

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"encoding/json"
	"errors"
	"fmt"
	"gnd.la/util/textutil"
	"net/http"
	"net/url"
	"strings"
)

var (
	signinUrl             = "https://www.pinterest.com/resource/UserSessionResource/create/"
	createPinUrl          = "http://www.pinterest.com/resource/PinResource/create/"
	noopUrl               = "http://www.pinterest.com/resource/NoopResource/get/"
	sessionCookieName     = "_pinterest_sess"
	csrfCookieName        = "csrftoken"
	errNoCookie           = errors.New("can't find session cookie")
	errUnexpectedResponse = errors.New("unexpected response from Pinterest")
)

type Account struct {
	Username string
	Password string
}

func (a *Account) Parse(raw string) error {
	fields, err := textutil.SplitFieldsOptions(raw, ":", &textutil.SplitOptions{ExactCount: 2})
	if err != nil {
		return err
	}
	a.Username = fields[0]
	a.Password = fields[1]
	return nil
}

type Session struct {
	Id      string
	Csrf    string
	Version string
	Account *Account
}

type Board struct {
	Id   string
	Name string
}

type Pin struct {
	Id          string
	Link        string
	Image       string
	Description string
}

func request(session *Session, u string, method string, ref string, data map[string]interface{}) (*http.Response, error) {
	var form *url.Values
	if data != nil {
		if session != nil && session.Version != "" {
			context, _ := data["context"].(map[string]interface{})
			if context == nil {
				context = map[string]interface{}{}
				data["context"] = context
			}
			context["app_version"] = session.Version
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		form = &url.Values{"data": []string{string(jsonData)}}
	}
	var req *http.Request
	var err error
	switch method {
	case "GET":
		if form != nil {
			req, err = http.NewRequest("GET", u+"?"+form.Encode(), nil)
		} else {
			req, err = http.NewRequest("GET", u, nil)
		}
	case "POST":
		if form != nil {
			req, err = http.NewRequest("POST", u, strings.NewReader(form.Encode()))
		} else {
			req, err = http.NewRequest("POST", u, nil)
		}
		if req != nil && form != nil {
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		}
	}
	if err != nil {
		return nil, err
	}
	req.Header.Add("Origin", "https://www.pinterest.com")
	if ref != "" {
		req.Header.Add("Referer", ref)
	}
	req.Header.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Add("Accept-Language", "en-US,en;q=0.8")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	if req.Header.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
	}
	req.Header.Add("X-NEW-APP", "1")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Safari/537.36")
	if session != nil {
		if session.Id != "" {
			req.AddCookie(&http.Cookie{
				Name:  sessionCookieName,
				Value: session.Id,
			})
		}
		if session.Csrf != "" {
			req.AddCookie(&http.Cookie{
				Name:  csrfCookieName,
				Value: session.Csrf,
			})
			req.Header.Add("X-CSRFToken", session.Csrf)
		}
		req.AddCookie(&http.Cookie{
			Name:  "_track_cm",
			Value: "1",
		})
	}
	client := &http.Client{}
	return client.Do(req)
}

func getCookie(resp *http.Response, name string) string {
	for _, v := range resp.Cookies() {
		if v.Name == name {
			return v.Value
		}
	}
	return ""
}

func parseJson(resp *http.Response) (map[string]interface{}, error) {
	dec := json.NewDecoder(resp.Body)
	var m map[string]interface{}
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if rresp, ok := m["resource_response"].(map[string]interface{}); ok {
		if err, ok := rresp["error"].(map[string]interface{}); ok {
			msg, _ := err["message"].(string)
			if msg == "" {
				msg, _ = err["code"].(string)
			}
			return nil, errors.New(msg)
		}
	}
	return m, nil
}

func nodeAttr(n *html.Node, key string) string {
	for _, v := range n.Attr {
		if v.Key == key {
			return v.Val
		}
	}
	return ""
}

func nodeText(n *html.Node) string {
	var buf bytes.Buffer
	_nodeText(n, &buf)
	return buf.String()
}

func _nodeText(n *html.Node, b *bytes.Buffer) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_nodeText(c, b)
	}
}

func getSession() (*Session, error) {
	resp, err := request(nil, "https://www.pinterest.com", "GET", "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	sess := getCookie(resp, sessionCookieName)
	csrf := getCookie(resp, csrfCookieName)
	if sess == "" || csrf == "" {
		return nil, errNoCookie
	}
	return &Session{
		Id:      sess,
		Csrf:    csrf,
		Version: resp.Header.Get("Pinterest-Version"),
	}, nil
}

func SignIn(account *Account) (*Session, error) {
	sess, err := getSession()
	if err != nil {
		return nil, err
	}
	resp, err := request(sess, signinUrl, "POST", "https://www.pinterest.com", map[string]interface{}{
		"options": map[string]interface{}{
			"username_or_email": account.Username,
			"password":          account.Password,
		},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_, err = parseJson(resp)
	if err != nil {
		return nil, err
	}
	sess.Account = account
	sess.Id = getCookie(resp, sessionCookieName)
	return sess, nil
}

func (s *Session) Boards() ([]*Board, error) {
	data := map[string]interface{}{
		"options": map[string]string{},
		"module": map[string]interface{}{
			"name":    "PinCreate",
			"options": map[string]interface{}{},
			"append":  false,
		},
	}
	resp, err := request(s, noopUrl, "GET", "", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	m, err := parseJson(resp)
	if err != nil {
		return nil, err
	}
	if module, ok := m["module"].(map[string]interface{}); ok {
		if markup, ok := module["html"].(string); ok {
			doc, err := html.Parse(strings.NewReader(markup))
			if err != nil {
				return nil, err
			}
			var boards []*Board
			var f func(*html.Node)
			f = func(n *html.Node) {
				if n.Type == html.ElementNode {
					if n.Data == "li" && nodeAttr(n, "class") == "boardPickerItem" {
						id := nodeAttr(n, "data-id")
						name := strings.TrimSpace(nodeText(n))
						boards = append(boards, &Board{
							Id:   id,
							Name: name,
						})
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			f(doc)
			return boards, nil
		}
	}
	return nil, errUnexpectedResponse
}

func (s *Session) Post(board *Board, pin *Pin) (*Pin, error) {
	var source string
	if pin.Link != "" {
		source = pin.Link
	} else {
		source = pin.Image
	}
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	host := url.QueryEscape(fmt.Sprintf("%s://%s", u.Scheme, u.Host))
	ref := fmt.Sprintf("http://www.pinterest.com/pin/find/?url=%s", host)
	resp, err := request(s, createPinUrl, "POST", ref, map[string]interface{}{
		"options": map[string]interface{}{
			"link":           pin.Link,
			"is_video":       nil,
			"image_url":      pin.Image,
			"method":         "scraped",
			"description":    pin.Description,
			"share_twitter":  false,
			"share_facebook": false,
			"board_id":       board.Id,
		},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	m, err := parseJson(resp)
	if err != nil {
		return nil, err
	}
	if rresp, ok := m["resource_response"].(map[string]interface{}); ok {
		if data, ok := rresp["data"].(map[string]interface{}); ok {
			if rtype, ok := data["type"].(string); ok && rtype == "pin" {
				if id, ok := data["id"].(string); ok {
					pin.Id = id
					return pin, nil
				}
			}
		}
	}
	return nil, errUnexpectedResponse
}
