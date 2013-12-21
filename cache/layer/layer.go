package layer

import (
	"encoding/gob"
	"errors"
	"gnd.la/cache"
	"gnd.la/encoding/codec"
	"gnd.la/log"
	"gnd.la/mux"
	"net/http"
)

var (
	fromLayer     = []string{"true"}
	layerCodec    = codec.Get("gob")
	errNoCache    = errors.New("nil cache passed to cache layer")
	errNoMediator = errors.New("nil mediator passed to cache layer")
)

type cachedResponse struct {
	Header     http.Header
	StatusCode int
	Data       []byte
}

// Layer allows caching complete responses to requests.
// Use New to initialize a Layer.
type Layer struct {
	cache    *cache.Cache
	mediator Mediator
}

// New returns a new layer, returning only errors if
// the cache or the mediator are nil
func New(c *cache.Cache, m Mediator) (*Layer, error) {
	if c == nil {
		return nil, errNoCache
	}
	if m == nil {
		return nil, errNoMediator
	}
	return &Layer{cache: c, mediator: m}, nil
}

// Cache returns the Layer's cache.
func (la *Layer) Cache() *cache.Cache {
	return la.cache
}

// Mediator returns the Layer's mediator.
func (la *Layer) Mediator() Mediator {
	return la.mediator
}

// Wrap takes a mux.Handler and returns a new mux.Handler
// wrapped by the Layer. Responses will be cached according
// to what the Layer's Mediator indicates.
func (la *Layer) Wrap(handler mux.Handler) mux.Handler {
	return func(ctx *mux.Context) {
		if la.mediator.Skip(ctx) {
			handler(ctx)
			return
		}
		key := la.mediator.Key(ctx)
		data, _ := la.cache.GetBytes(key)
		if data != nil {
			// has cached data
			var response *cachedResponse
			err := layerCodec.Decode(data, &response)
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
		if la.mediator.Cache(ctx, w.statusCode, w.header) {
			response := &cachedResponse{w.header, w.statusCode, w.buf.Bytes()}
			data, err := layerCodec.Encode(response)
			if err == nil {
				ctx.SetCached(true)
				expiration := la.mediator.Expires(ctx, w.statusCode, w.header)
				la.cache.SetBytes(key, data, expiration)
			} else {
				log.Errorf("Error encoding cached response: %v", err)
			}
		}
	}
}

func init() {
	gob.Register(&cachedResponse{})
}
