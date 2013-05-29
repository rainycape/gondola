package layer

import (
	"bytes"
	"encoding/gob"
	"gondola/cache"
	"gondola/cache/codec"
	"gondola/mux"
	"gondola/util"
	"net/http"
)

var (
	DefaultExpiration = 600
)

type Func func(mux.Handler) mux.Handler
type KeyFunc func(*mux.Context) string
type FilterFunc func(*mux.Context, int, http.Header) bool
type ExpirationFunc func(*mux.Context, int, http.Header) int

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
		return func(ctx *mux.Context) {
			key := k(ctx)
			if key != "" {
				data := c.GetBytes(key)
				if data != nil {
					var response *cachedResponse
					err := codec.GobCodec.Decode(data, &response)
					if err == nil {
						ctx.SetServedFromCache(true)
						header := ctx.Header()
						for k, v := range response.Header {
							header[k] = v
						}
						ctx.WriteHeader(response.StatusCode)
						ctx.Write(response.Data)
						return
					}
				}
				rw := ctx.ResponseWriter
				lw := &layerWriter{ctx.ResponseWriter, bytes.NewBuffer(nil), 0, nil}
				ctx.ResponseWriter = lw
				fun(ctx)
				ctx.ResponseWriter = rw
				if f(ctx, lw.statusCode, lw.header) {
					response := &cachedResponse{lw.header, lw.statusCode, lw.buf.Bytes()}
					data, err := codec.GobCodec.Encode(response)
					if err == nil {
						ctx.SetCached(true)
						c.SetBytes(key, data, e(ctx, lw.statusCode, lw.header))
					}
				}
			} else {
				fun(ctx)
			}
		}
	}
}

func DefaultKeyFunc(ctx *mux.Context) string {
	r := ctx.R
	if r.Method == "GET" {
		return util.Md5([]byte(r.URL.String()))
	}
	return ""
}

func DefaultFilterFunc(ctx *mux.Context, code int, header http.Header) bool {
	return code == http.StatusOK
}

func DefaultExpirationFunc(ctx *mux.Context, code int, header http.Header) int {
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
