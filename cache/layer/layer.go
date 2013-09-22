package layer

import (
	"encoding/gob"
	"errors"
	"gnd.la/cache"
	"gnd.la/cache/codec"
	"gnd.la/log"
	"gnd.la/mux"
	"net/http"
)

var (
	fromLayer = []string{"true"}
)

type cachedResponse struct {
	Header     http.Header
	StatusCode int
	Data       []byte
}

func New(c *cache.Cache, m Mediator) mux.Transformer {
	if c == nil {
		c = cache.NewDefault()
	}
	if m == nil {
		panic(errors.New("nil mediator passed to cache layer"))
	}
	return func(handler mux.Handler) mux.Handler {
		return func(ctx *mux.Context) {
			if m.Skip(ctx) {
				handler(ctx)
				return
			}
			key := m.Key(ctx)
			data := c.GetBytes(key)
			if data != nil {
				// has cached data
				var response *cachedResponse
				err := codec.GobCodec.Decode(data, &response)
				if err == nil {
					ctx.SetServedFromCache(true)
					header := ctx.Header()
					for k, v := range response.Header {
						header[k] = v
					}
					header["X-Gondola-From-Layer"] = fromLayer
					ctx.WriteHeader(response.StatusCode)
					ctx.Write(response.Data)
					return
				}
			}

			rw := ctx.ResponseWriter
			w := newWriter(rw)
			ctx.ResponseWriter = w
			handler(ctx)
			ctx.ResponseWriter = rw
			if m.Cache(ctx, w.statusCode, w.header) {
				response := &cachedResponse{w.header, w.statusCode, w.buf.Bytes()}
				data, err := codec.GobCodec.Encode(response)
				if err == nil {
					ctx.SetCached(true)
					expiration := m.Expires(ctx, w.statusCode, w.header)
					c.SetBytes(key, data, expiration)
				} else {
					log.Errorf("Error encoding cached response: %v", err)
				}
			}
		}
	}
}

func init() {
	gob.Register(&cachedResponse{})
}
