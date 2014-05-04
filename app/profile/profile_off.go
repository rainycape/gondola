// +build !profile appengine

package profile

const profileIsOn = false

func goroutineId() int32 {
	return -1
}
