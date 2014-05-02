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
			timeout := tr.timeout
			if timeout == 0 {
				return net.Dial(network, addr)
			}
			start := time.Now()
			conn, err := net.DialTimeout(network, addr, timeout)
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(start.Add(timeout))
			return conn, nil
		},
	}}
}

func (t *transport) Timeout() time.Duration {
	return t.timeout
}

func (t *transport) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}
