// +build appengine

package formula

func bint(b bool) int {
	if b {
		return 1
	}
	return 0
}
