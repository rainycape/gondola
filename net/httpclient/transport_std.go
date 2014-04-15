// +build !appengine

package httpclient

import (
	"net"
	"net/http"
	"time"
)

func newRoundTripper(ctx Context, tr *transport) http.RoundTripper {
	return &http.Transport{
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
	}
}

func (t *transport) Deadline() time.Duration {
	return t.deadline
}

func (t *transport) SetDeadline(deadline time.Duration) {
	t.deadline = deadline
}
