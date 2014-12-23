// Package pagination implements helper functions for request handlers
// which present results organized in numbered pages.
//
// This package can only be used with named handlers (registered via
// app.App.HandleNamed) and having patterns with no optional groups.
// (e.g. no "(something)?").
//
// Paging in this package assumes the following:
//
//  - Pages start at 1.
//  - The first page URL does not include the page number.
//  - Other pages end with someprefix-<page-number>-somesuffix (see Pattern).
//  - A URL with the page number set to 0 or 1 produces a redirect.
//  - A URL with the page number > the available number of pages produces a 404.
//
// A typical handler using this package would look something like this.
//
//  Registered as: App.HandleNamed("/popular/" + pagination.Pattern("", "/") + "$", PopularHandler, "popular")
//
//  func PopularHandler(ctx *app.Context) {
//	const itemsPerPage = 15
//	q := ctx.Orm().Query(... some conditions ...)
//	count := q.Table(o.NameTable("Item")).MustCount()
//	p := pagination.New(ctx, int(count), itemsPerPage)
//	if p == nil {
//	    // Request had an invalid page number and has been
//	    // already served
//	    return
//	}
//	var items []*Item
//	p.Page(p.Page(), itemsPerPage).MustAll(&items)
//	data := map[string]interface{}{
//	    "Items": items,
//	    "Page": p.Page(),
//	    "Paginator": p.Paginator(),
//	}
//	ctx.MustExecute("items.html", data)
//  }
package pagination
