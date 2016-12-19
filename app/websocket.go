package app

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/net/websocket"
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
