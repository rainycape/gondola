package main

import (
	"bytes"
	"compress/flate"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/howeyc/gopass"
	"gnd.la/admin"
	"gnd.la/app"
	"gnd.la/app/profile"
	"gnd.la/crypto/cryptoutil"
	"gnd.la/encoding/base64"
	"gnd.la/log"
	"gnd.la/util/stringutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

var (
	errAuthRequired = errors.New("authentication required")
	errAuthFailed   = errors.New("authentication failed")
)

// copied from gnd.la/app/profile.go, keep in sync
type profileInfo struct {
	Elapsed time.Duration     `json:"e"`
	Timings []*profile.Timing `json:"t"`
}

func requestProfile(u string, method string, values url.Values, secret string) (*profileInfo, error) {
	var req *http.Request
	var err error
	if method == "POST" {
		if len(values) > 0 {
			req, err = http.NewRequest(method, u, strings.NewReader(values.Encode()))
			if req != nil {
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			}
		} else {
			req, err = http.NewRequest(method, u, nil)
		}
	} else {
		reqURL := u
		if len(values) > 0 {
			reqURL += "?" + values.Encode()
		}
		req, err = http.NewRequest(method, reqURL, nil)
	}
	if err != nil {
		return nil, err
	}
	if secret != "" {
		ts := time.Now().Unix()
		nonce := stringutil.Random(32)
		signer := cryptoutil.Signer{Salt: []byte(profile.Salt), Key: []byte(secret)}
		signed, err := signer.Sign([]byte(fmt.Sprintf("%d:%s", ts, nonce)))
		if err != nil {
			return nil, err
		}
		req.Header.Add(profile.HeaderName, signed)
	} else {
		req.Header.Add(profile.HeaderName, "true")
	}
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	log.Debugf("Request: \n%s", string(dump))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	value := resp.Header.Get(profile.HeaderName)
	switch value {
	case "":
		return nil, fmt.Errorf("no profiling info on %s - is profiling enabled?", u)
	case "auth":
		return nil, errAuthRequired
	case "denied":
		return nil, errAuthFailed
	}
	decoded, err := base64.Decode(value)
	if err != nil {
		return nil, err
	}
	r := flate.NewReader(bytes.NewReader(decoded))
	defer r.Close()
	dec := json.NewDecoder(r)
	var info *profileInfo
	if err := dec.Decode(&info); err != nil {
		return nil, err
	}
	return info, nil
}

func Profile(ctx *app.Context) {
	u := ctx.IndexValue(0)
	if u == "" {
		log.Fatalf("url can't be empty")
	}
	var method string
	ctx.ParseParamValue("m", &method)
	var data string
	ctx.ParseParamValue("d", &data)
	var values url.Values
	if data != "" {
		values = make(url.Values)
		fields, err := stringutil.SplitFields(data, ";")
		if err != nil {
			log.Fatalf("invalid data: %s", err)

		}
		for _, v := range fields {
			parts, err := stringutil.SplitFields(data, ";")
			if err != nil || len(parts) != 2 {
				log.Fatalf("invalid data parameter %q: %s", v, err)
			}
			values.Add(parts[0], parts[1])
		}
	}
	parsed, err := url.Parse(u)
	if err != nil {
		log.Fatalf("invalid url %q: %s", u, err)
	}
	host := parsed.Host
	var secret string
	var info *profileInfo
	for {
		info, err = requestProfile(u, method, values, secret)
		if err == nil {
			break
		}
		if err == errAuthRequired {
			fmt.Printf("Enter secret for %s: ", host)
			secret = string(gopass.GetPasswd())
			fmt.Println("")
			continue
		}
		if err == errAuthFailed {
			fmt.Printf("Incorrect secret\nEnter secret for %s: ", host)
			secret = string(gopass.GetPasswd())
			fmt.Println("")
			continue
		}
		log.Fatal(err)
	}
	width := 80
	fmt.Printf("total %s\n%s\n\n", info.Elapsed, strings.Repeat("=", width))
	other := info.Elapsed
	for _, v := range info.Timings {
		other -= v.Total()
		fmt.Printf("%s - %d events - %s\n%s\n", v.Name, v.Count(), v.Total(), strings.Repeat("-", width))
		maxLength := 0
		for _, ev := range v.Events {
			if length := len(fmt.Sprintf("%s", ev.Elapsed())); length > maxLength {
				maxLength = length
			}
		}
		for ii, ev := range v.Events {
			notesWidth := width - maxLength - 6
			notes := formatNotes(ev.Notes, notesWidth)
			fmt.Printf("| %s | %s |\n", pad(fmt.Sprintf("%s", ev.Elapsed()), maxLength), pad(notes[0], notesWidth))
			for _, n := range notes[1:] {
				fmt.Printf("| %s | %s |\n", pad("", maxLength), pad(n, notesWidth))
			}
			if ii < len(v.Events)-1 {
				fmt.Println(strings.Repeat("-", width))
			}
		}
		fmt.Printf("%s\n\n", strings.Repeat("=", width))
	}
	fmt.Printf("others - %s\n", other)
}

func pad(s string, width int) string {
	if len(s) < width {
		return s + strings.Repeat(" ", width-len(s))
	}
	return s
}

func formatNotes(notes []*profile.Note, width int) []string {
	if len(notes) == 0 {
		return []string{""}
	}
	var output []string
	for _, v := range notes {
		text := fmt.Sprintf("%s | %s", v.Title, v.Text)
		for _, line := range strings.Split(text, "\n") {
			if len(line) <= width {
				output = append(output, line)
				continue
			}
			rem := line
			for {
				if len(rem) <= width {
					output = append(output, rem)
					break
				}
				cur := rem[:width]
				if p := strings.LastIndexAny(cur, " ,."); p > width/2 {
					cur = cur[:p+1]
				}
				output = append(output, cur)
				rem = rem[len(cur):]
			}
		}
	}
	return output
}

func init() {
	admin.Register(Profile, &admin.Options{
		Help: "Shows profiling information for a remote server running a Gondola app",
		Flags: admin.Flags(
			admin.StringFlag("m", "GET", "HTTP method"),
			admin.StringFlag("d", "", "Data to be sent in the request in the form k1=v1;k2=v2..."),
		),
	})
}
