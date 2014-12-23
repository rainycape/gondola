package pagination

import (
	"gnd.la/app"
	"gnd.la/html/paginator"
)

// Pattern returns the pattern required to append to
// an existing pattern in order to perform pagination
// using this package. The prefix argument will be
// part of the URL just before the page.
//
// For example, with a handler pattern like "/popular/"
//
//  - Pattern("", "/") will generate pages like "/popular/7/"
//  - Pattern("", "") will generate pages like "/popular/7"
//  - Pattern("page-", "/") will generate pages like "/popular/page-7/"
//
// To register a handler with the pattern, someone might write:
//
//  myapp.HandleNamed("/popular/" + pagination.Pattern("", "/") + "$", PopularHandler, "popular")
//
// Where myapp is a gnd.la/app.App instance.
func Pattern(prefix string, suffix string) string {
	return "(?:" + prefix + "(?P<page>\\d+)" + suffix + ")?"
}

// Pagination is an opaque type encapsulating the pagination data
// Callers should just create a new Pagination from an app.Context
// using New and retrieve the current page using its Pagination.Page
// method.
type Pagination struct {
	count   int
	perPage int
	page    int
	pager   *Pager
	ctx     *app.Context
}

// New returns a new *Pagination for the given app.Context, item count and
// number of items per page. If there's a programming error, this function
// will panic. If this function returns nil, an invalid page was specified
// e.g. (user typed a URL with the 0 or 1 page number) and you should
// consider the request as served and not write anything else to the
// app.Context.
func New(ctx *app.Context, itemCount int, itemsPerPage int) *Pagination {
	p, err := newPagination(ctx, itemCount, itemsPerPage)
	if err != nil {
		panic(err)
	}
	if !p.isValid() {
		return nil
	}
	return p
}

// Page returns the current page number. Note that pages start at 1.
func (p *Pagination) Page() int {
	return p.page
}

// Paginator returns a new *paginator.Paginator prepared to
// be rendered by a template.
func (p *Pagination) Paginator() *paginator.Paginator {
	return paginator.New(p.count, p.perPage, p.page, p.pager)
}

func newPagination(ctx *app.Context, itemCount int, itemsPerPage int) (*Pagination, error) {
	pager, err := NewPager(ctx)
	if err != nil {
		return nil, err
	}
	return &Pagination{
		count:   itemCount,
		perPage: itemsPerPage,
		pager:   pager,
		ctx:     ctx,
	}, nil
}

func (p *Pagination) isValid() bool {
	page := -1
	if err := p.ctx.ParseParamValue("page", &page); err != nil {
		if _, ok := err.(*app.InvalidParameterError); ok {
			// page was provided but the regexp must have
			// been wrong since it couldn't not be parsed
			// as an int. panic here so the error is
			// easier to spot.
			panic(err)
		}
	} else if page == 0 || page == 1 {
		p.ctx.Redirect(p.pager.URL(1), true)
		return false
	}
	lastPage := ((p.count - 1) / p.perPage) + 1
	if page > lastPage {
		p.ctx.NotFound(p.ctx.Tc("paginator", "page not found"))
		return false
	}
	if page < 0 {
		page = 1
	}
	p.page = page
	return true
}
