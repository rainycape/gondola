package formutil

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/form"
	"gnd.la/html"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var (
	css = "#recaptcha_area, #recaptcha_table { line-height: 16px !important;}"
)

// Reaptcha provides a captcha using the reCAPTCHA API. The theme parameter can
// be one of "red", "blackglass", "white" and "clean". Any other value (including
// the empty string) results in the red theme.
func ReCaptcha(publicKey string, privateKey string, theme string) interface{} {
	return &reCaptcha{
		publicKey:  publicKey,
		privateKey: privateKey,
		theme:      theme,
	}
}

type reCaptcha struct {
	publicKey  string
	privateKey string
	theme      string
	msg        string
	Recaptcha  string `form:"recaptcha_challenge_field,hidden,optional"`
}

func (r *reCaptcha) FieldAddOns(ctx *app.Context, field *form.Field) []*form.AddOn {
	if field.GoName == "Recaptcha" {
		style := &html.Node{
			Tag:      "style",
			Attrs:    html.Attrs{"type": "text/css"},
			Children: html.Text(css),
		}
		theme := &html.Node{
			Tag:      "script",
			Attrs:    html.Attrs{"type": "text/javascript"},
			Children: html.Text(fmt.Sprintf("var RecaptchaOptions = {theme : '%s'};", r.theme)),
		}
		src := fmt.Sprintf("//www.google.com/recaptcha/api/challenge?k=%s", r.publicKey)
		if r.msg != "" {
			src += fmt.Sprintf("&error=%s", r.msg)
		}
		script := &html.Node{
			Tag:   "script",
			Attrs: html.Attrs{"type": "text/javascript", "src": src},
		}
		return []*form.AddOn{
			&form.AddOn{
				Node: html.Div(style, theme, script),
			},
		}
	}
	return nil
}

func (r *reCaptcha) ValidateRecaptcha(ctx *app.Context) error {
	valid, msg := r.responseIsValid(ctx)
	r.msg = msg
	if valid {
		return nil
	}
	return fmt.Errorf("incorrect")
}

func (r *reCaptcha) responseIsValid(ctx *app.Context) (bool, string) {
	challenge := ctx.FormValue("recaptcha_challenge_field")
	response := ctx.FormValue("recaptcha_response_field")
	values := url.Values{
		"privatekey": {r.privateKey},
		"remoteip":   {ctx.RemoteAddress()},
		"challenge":  {challenge},
		"response":   {response},
	}
	resp, err := http.PostForm("http://www.google.com/recaptcha/api/verify", values)
	if err == nil {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			lines := strings.Split(string(b), "\n")
			if len(lines) > 0 {
				if lines[0] == "true" {
					return true, ""
				}
				if len(lines) > 1 {
					return false, lines[1]
				}
			}
		}
	}
	return false, ""
}
