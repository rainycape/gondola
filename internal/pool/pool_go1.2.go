// +build !go1.3

package pool

import (
	"runtime"
)

type Pool struct {
	ch  chan interface{}
	New func() interface{}
}

// New returns a new Pool. The size argument is
// ignored on Go >= 1.3.
func New(size int) *Pool {
	if size == 0 {
		size = runtime.GOMAXPROCS(0) * 2
	}
	return &Pool{ch: make(chan interface{}, size)}
}

func (p *Pool) Get() interface{} {
	select {
	case x := <-p.ch:
		return x
	default:
	}
	if p.New != nil {
		return p.New()
	}
	return nil
}

func (p *Pool) Put(x interface{}) {
	select {
	case p.ch <- x:
	default:
	}
}
