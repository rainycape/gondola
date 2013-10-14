package social

import (
	"gnd.la/mux"
	"time"
)

type ShareProvider interface {
	LastShare(ctx *mux.Context, service Service) (time.Time, error)
	Item(ctx *mux.Context, service Service) (*Item, error)
	Shared(ctx *mux.Context, service Service, item *Item, result interface{}, err error)
}
