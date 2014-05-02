// +build appengine

package sql

func stobs(s string) []byte {
	return []byte(s)
}

func bstos(b []byte) string {
	return string(b)
}
