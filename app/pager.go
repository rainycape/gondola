package app

import (
	"errors"

	"gnd.la/html/paginator"
)

type pager struct {
	name   string
	params []interface{}
	ctx    *Context
}

func (p *pager) URL(page int) string {
	params := p.params
	if page != 1 {
		params = append(params, page)
	}
	return p.ctx.MustReverse(p.name, params...)
}

// Pager returns a pager which can be used with a paginator.Paginator.
// In order for the automatic pager to work, the request must be served
// by a named handler (so the Pager can call ctx.Reverse), use only
// named capture parameters in its pattern and have the last parameter
// be optional and named page (which indicates the page number). e.g.
// a handler like this would work.
//
//  App.HandleNamed("^/c/(?P<cat_id>\\d+)/(?:(?P<page>\\d+)/)?$", ListHandler, "category")
//
// Then the handler would need to retrieve the page number and use it to create
// the paginator.
//
//  const itemsPerPage = 42
//  var page int
//  ctx.ParseParamValue("page", &page)
//  if page <= 0 {
//	page = 1
//  }
//  count, err := ...
//  items, err := ...
//  p := paginator.New(int(count+1)/itemsPerPage, page, ctx.Pager())
//
func (c *Context) Pager() paginator.Pager {
	p, err := c.pager()
	if err != nil {
		panic(err)
	}
	return p
}

func (c *Context) pager() (paginator.Pager, error) {
	name := c.HandlerName()
	if name == "" {
		return nil, errors.New("can't generate a pager from an unnamed handler")
	}
	params := c.Params()
	if len(params) == 0 {
		return nil, errors.New("can't generate a pager from a handler with no parameters, must have a page parameter")
	}
	if params[len(params)-1] != "page" {
	}
	reverseParams := make([]interface{}, len(params)-1)
	for ii, v := range params[:len(params)-1] {
		reverseParams[ii] = c.ParamValue(v)
	}
	return &pager{
		name:   name,
		params: reverseParams,
		ctx:    c,
	}, nil
}
