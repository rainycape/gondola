package social

import (
	"gnd.la/app"
	"time"
)

type ShareProvider interface {
	LastShare(ctx *app.Context, service Service) (time.Time, error)
	Item(ctx *app.Context, service Service) (*Item, error)
	Shared(ctx *app.Context, service Service, item *Item, result interface{}, err error)
}
