package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/websocket"

	"gnd.la/internal/devutil/devserver"
)

type wsKey int

const (
	ctxKey wsKey = iota
	websocketKey
)

// HandleWebsocket has the same semantics as App.Handle, but responds to websocket
// requests rather than normal HTTP(S) requests. Use Context.Websocket to retrieve
// the *websocket.Conn in the Handler.
func (app *App) HandleWebsocket(pattern string, handler Handler, opts ...HandlerOption) {
	if handler == nil {
		panic(fmt.Errorf("handler for websocket pattern %q can't be nil", pattern))
	}
	wsHandler := websocket.Handler(func(ws *websocket.Conn) {
		ctx := ws.Request().Context().Value(ctxKey).(*Context)
		ctx.Set(websocketKey, ws)
		handler(ctx)
	})
	reqHandler := func(ctx *Context) {
		req := ctx.Request()
		newCtx := context.WithValue(req.Context(), ctxKey, ctx)
		newReq := ctx.Request().WithContext(newCtx)
		rw := ctx.ResponseWriter
		ctx.ResponseWriter = nil
		wsHandler.ServeHTTP(rw, newReq)
	}
	app.Handle(pattern, reqHandler, opts...)
}

// Websocket returns the *websocket.Conn assocciated with the current request. If the handler
// wasn't added with App.HandleWebsocket, this function will panic.
func (c *Context) Websocket() *websocket.Conn {
	ws, ok := c.Get(websocketKey).(*websocket.Conn)
	if !ok {
		panic(errors.New("no websocket assocciated with handler - did you use App.Handle() rather than App.HandleWebsocket()?"))
	}
	return ws
}

// WebsocketURL returns an absolute websocket URL from a relative
// URL (relative to the current request), adjusting also the protocol
// (e.g. http to ws and https to wss). Note that this function calls
// Context.URL(), so check its documentation to make sure the current
// request URL can be correctly determined.
func (c *Context) WebsocketURL(rel string) (*url.URL, error) {
	u, err := c.URL().Parse(rel)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}
	if sep := strings.IndexByte(u.Host, ':'); sep >= 0 && devserver.IsActive() {
		// If we're running from the development server, we need to send
		// the request directly to the app, since the httputil proxy doesn't
		// support websockets.
		u.Host = u.Host[:sep] + ":" + strconv.Itoa(c.app.Config().Port)
	}
	return u, nil
}
