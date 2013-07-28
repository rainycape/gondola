// +build appengine !amd64 !linux

package formula

func vmJit(p program) (Formula, error) {
	return nil, errJitNotSupported
}
