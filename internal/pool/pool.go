// +build go1.3

package pool

import (
	"sync"
)

type Pool sync.Pool

// New returns a new Pool. The size argument is
// ignored on Go >= 1.3.
func New(size int) *Pool {
	return &Pool{}
}

func (p *Pool) Get() interface{} {
	return (*sync.Pool)(p).Get()
}

func (p *Pool) Put(x interface{}) {
	(*sync.Pool)(p).Put(x)
}
