package driver

import (
	"fmt"
	"strings"
)

// DefaultPort adds port to the addr passed in
// if it doesn't already have a port. It works for
// hostnames, IPv4 and IPv6 addresses. In case addr
// is empty, localhost:port is returned.
func DefaultPort(addr string, port int) string {
	if addr == "" {
		return fmt.Sprintf("localhost:%d", port)
	}
	requires := false
	brackets := false
	c := strings.Count(addr, ":")
	v6 := c > 1
	if c == 0 {
		requires = true
	} else if c > 1 {
		if strings.HasPrefix(addr, "[") {
			brackets = true
			brk := strings.Index(addr, "]")
			if brk < 0 {
				addr = addr[1:]
				requires = true
				brackets = false
			} else {
				requires = !strings.Contains(addr[brk:], ":")
			}
		} else {
			requires = true
		}
	}
	if requires {
		if v6 && !brackets {
			return fmt.Sprintf("[%s]:%d", addr, port)
		}
		return fmt.Sprintf("%s:%d", addr, port)
	}
	return addr
}
