// +build !appengine

package httpclient

import (
	"net"
	"net/http"
	"time"
)

type roundTripper struct {
	http.Transport
}

func (r *roundTripper) Proxy() Proxy {
	return r.Transport.Proxy
}

func (r *roundTripper) SetProxy(proxy Proxy) {
	r.Transport.Proxy = proxy
}

func newRoundTripper(ctx Context, tr *transport) http.RoundTripper {
	return &roundTripper{http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			deadline := tr.deadline
			if deadline == 0 {
				return net.Dial(network, addr)
			}
			start := time.Now()
			conn, err := net.DialTimeout(network, addr, deadline)
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(start.Add(deadline))
			return conn, nil
		},
	}}
}

func (t *transport) Deadline() time.Duration {
	return t.deadline
}

func (t *transport) SetDeadline(deadline time.Duration) {
	t.deadline = deadline
}
