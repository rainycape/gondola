package app

import "testing"
import "net/http"
import "io/ioutil"
import "bytes"
import "strconv"
import "compress/gzip"

import "compress/zlib"

func TestBodyReader(t *testing.T) {
	const (
		payload = "this is a test"
	)
	type contentEncoding struct {
		Name    string
		Encoder func([]byte) ([]byte, error)
	}
	identity := func(data []byte) ([]byte, error) { return data, nil }
	encodings := []contentEncoding{
		{
			"gzip",
			func(data []byte) ([]byte, error) {
				var buf bytes.Buffer
				zw := gzip.NewWriter(&buf)
				if _, err := zw.Write(data); err != nil {
					return nil, err
				}
				zw.Close()
				return buf.Bytes(), nil
			},
		},
		{
			"deflate",
			func(data []byte) ([]byte, error) {
				var buf bytes.Buffer
				zw := zlib.NewWriter(&buf)
				if _, err := zw.Write(data); err != nil {
					return nil, err
				}
				zw.Close()
				return buf.Bytes(), nil

			},
		},
		{
			"identity",
			identity,
		},
		{
			"",
			identity,
		},
		{
			"not-a-valid-content-encoding",
			identity,
		},
	}

	for _, e := range encodings {
		e := e
		t.Run(e.Name, func(t *testing.T) {
			t.Parallel()
			encoded, err := e.Encoder([]byte(payload))
			if err != nil {
				t.Errorf("error encoding data: %v", err)
				return
			}
			req := &http.Request{
				Body:   ioutil.NopCloser(bytes.NewReader(encoded)),
				Header: make(http.Header),
			}
			req.ContentLength = int64(len(encoded))
			req.Header.Add("Content-Length", strconv.Itoa(len(encoded)))
			req.Header.Add("Content-Encoding", e.Name)
			ctx := &Context{
				R: req,
			}
			data, err := ioutil.ReadAll(ctx)
			if err != nil {
				t.Errorf("error reading from Context: %v", err)
				return
			}
			if s := string(data); s != payload {
				t.Errorf("expecting payload = %q, got %q instead", payload, s)
			}
		})
	}
}
