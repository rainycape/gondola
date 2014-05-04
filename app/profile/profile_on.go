// +build !appengine,profile

package profile

const profileIsOn = true

func goroutineId() int32

func init() {
	contexts.data = make(map[int32]*context)
}
