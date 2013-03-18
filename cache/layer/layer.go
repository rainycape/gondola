package layer

import (
	"bytes"
	"encoding/gob"
	"gondola/cache"
	"gondola/mux"
	"gondola/util"
	"net/http"
)

var (
	DefaultExpiration = 600
)

type Func func(mux.Handler) mux.Handler
type KeyFunc func(*http.Request, *mux.Context) string
type FilterFunc func(*http.Request, int, http.Header, *mux.Context) bool
type ExpirationFunc func(*http.Request, int, http.Header, *mux.Context) int

func New(c *cache.Cache, k KeyFunc, f FilterFunc, e ExpirationFunc) Func {
	if c == nil {
		c = cache.NewDefault()
	}
	if k == nil {
		k = DefaultKeyFunc
	}
	if f == nil {
		f = DefaultFilterFunc
	}
	if e == nil {
		e = DefaultExpirationFunc
	}
	return func(fun mux.Handler) mux.Handler {
		return func(w http.ResponseWriter, r *http.Request, ctx *mux.Context) {
			key := k(r, ctx)
			if key != "" {
				data := c.GetBytes(key)
				if data != nil {
					var response *cachedResponse
					err := cache.GobEncoder.Decode(data, &response)
					if err == nil {
						ctx.SetServedFromCache(true)
						header := w.Header()
						for k, v := range response.Header {
							header[k] = v
						}
						w.WriteHeader(response.StatusCode)
						w.Write(response.Data)
						return
					}
				}
				lw := &layerWriter{w, bytes.NewBuffer(nil), 0, nil}
				fun(lw, r, ctx)
				if f(r, lw.statusCode, lw.header, ctx) {
					response := &cachedResponse{lw.header, lw.statusCode, lw.buf.Bytes()}
					data, err := cache.GobEncoder.Encode(response)
					if err == nil {
						ctx.SetCached(true)
						c.SetBytes(key, data, e(r, lw.statusCode, lw.header, ctx))
					}
				}
			} else {
				fun(w, r, ctx)
			}
		}
	}
}

func DefaultKeyFunc(r *http.Request, ctx *mux.Context) string {
	if r.Method == "GET" {
		return util.Md5([]byte(r.URL.String()))
	}
	return ""
}

func DefaultFilterFunc(r *http.Request, code int, header http.Header, ctx *mux.Context) bool {
	return code == http.StatusOK
}

func DefaultExpirationFunc(r *http.Request, code int, header http.Header, ctx *mux.Context) int {
	return DefaultExpiration
}

type cachedResponse struct {
	Header     http.Header
	StatusCode int
	Data       []byte
}

type layerWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
	header     http.Header
}

func (lw *layerWriter) copyHeaders() {
	if lw.header == nil {
		lw.header = http.Header{}
		for k, v := range lw.ResponseWriter.Header() {
			lw.header[k] = v
		}
		if lw.statusCode == 0 {
			lw.statusCode = http.StatusOK
		}
	}
}

func (lw *layerWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.copyHeaders()
	lw.ResponseWriter.WriteHeader(code)
}

func (lw *layerWriter) Write(data []byte) (int, error) {
	n, err := lw.ResponseWriter.Write(data)
	if err == nil && n > 0 {
		lw.buf.Write(data)
		if lw.header == nil {
			lw.copyHeaders()
		}
	}
	return n, err
}

func init() {
	gob.Register(&cachedResponse{})
}
