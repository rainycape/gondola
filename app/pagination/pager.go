package pagination

import (
	"errors"
	"fmt"

	"gnd.la/app"
)

// Pager implements the paginator.Pager interface, using
// Context.Reverse to obtain the page URL. The handler pattern
// MUST end with an optional group which contains a captured
// group named "page". The rest of parameters MUST be mandatory and
// might or might not be named.
//
// In practice, this means the pattern must not have any optional
// groups except the one containing the page and must end with
// something like:
//
//  (?:(?P<page>\\d+)/)?$
//
// Note that the optional group might contain anything else. e.g.
// this pattern would also be valid.
//
//  (?:page-(?P<page>\\d+)/)?$
//
// One complete pattern might look like this:
//
//  App.HandleNamed("^/c/(\\d+)/(?:(?P<page>\\d+)/)?$", ListHandler, "category")
//
// The Parameters field must contain all values to be passed to
// Context.Reverse except the page number itself. The Pager will append the
// page number for each page != 1.
//
// Note that most users should not use this type directly, but rather use
// the more complete Pagination type from this package.
type Pager struct {
	// The handler name to reverse
	Name string
	// Parameters to pass to reverse *BEFORE* the page
	// number. The page number will be appended to this
	// slice, so it must be always the last parameter
	Parameters []interface{}
	// The current Context received in your handler.
	Ctx *app.Context
}

// NewPager returns a new *Pager instance. Note that the
// Context must satisfy the conditions mentioned in the
// Pager type documentation. Otherwise an error will be
// returned.
func NewPager(ctx *app.Context) (*Pager, error) {
	name := ctx.HandlerName()
	if name == "" {
		return nil, errors.New("can't generate a pager from an unnamed handler")
	}
	names := ctx.Provider().ParamNames()
	if len(names) == 0 {
		return nil, errors.New("can't generate a pager from a handler with no parameters, must have a page parameter")
	}
	if names[len(names)-1] != "page" {
		return nil, fmt.Errorf("last named parameter is %q, not page", names[len(names)-1])
	}
	params := make([]interface{}, ctx.Count()-1)
	for ii := 0; ii < ctx.Count()-1; ii++ {
		params[ii] = ctx.IndexValue(ii)
	}
	return &Pager{
		Name:       name,
		Parameters: params,
		Ctx:        ctx,
	}, nil
}

// URL returns the given page URL by using gnd.la/app.Context.Reverse.
func (p *Pager) URL(page int) string {
	params := p.Parameters
	if page != 1 {
		// Make sure we don't modify the initial
		// Parameters slice.
		newParams := make([]interface{}, len(params), len(params)+1)
		copy(newParams, params)
		params = append(newParams, page)
	}
	return p.Ctx.MustReverse(p.Name, params...)
}
