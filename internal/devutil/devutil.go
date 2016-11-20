package devutil

import (
	"io"
	"io/ioutil"
	"sync"

	"gnd.la/app"
	"gnd.la/internal/devutil/devserver"

	"time"

	"golang.org/x/net/websocket"
)

type broadcasterMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type Broadcaster struct {
	mu sync.RWMutex
	ws map[*websocket.Conn]struct{}
	ts time.Time
}

func (b *Broadcaster) BroadcastReload() {
	b.Broadcast(&broadcasterMessage{Type: "reload"})
}

func (b *Broadcaster) Broadcast(msg interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ws := range b.ws {
		websocket.JSON.Send(ws, msg)
	}
}

func (b *Broadcaster) Attach(a *app.App) {
	a.HandleWebsocket(b.pattern(), b.handler())
	a.AddTemplateVars(devserver.TemplateVars(&app.Context{}))
	a.AddTemplatePlugin(devserver.ReloadPlugin())
	b.ts = time.Now()
}

func (b *Broadcaster) pattern() string {
	return devserver.UpdatesPath()
}

func (b *Broadcaster) handler() func(*app.Context, *websocket.Conn) {
	return func(ctx *app.Context, ws *websocket.Conn) {
		b.add(ws)
		io.Copy(ioutil.Discard, ws)
		b.remove(ws)
	}
}

func (b *Broadcaster) add(ws *websocket.Conn) {
	b.mu.Lock()
	if b.ws == nil {
		b.ws = make(map[*websocket.Conn]struct{})
	}
	b.ws[ws] = struct{}{}
	b.mu.Unlock()
	websocket.JSON.Send(ws, &broadcasterMessage{
		Type:      "timestamp",
		Timestamp: b.ts.Unix(),
	})
}

func (b *Broadcaster) remove(ws *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.ws, ws)
}
