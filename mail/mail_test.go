package mail

import (
	"testing"
)

func testCredentials(t *testing.T, addr, server, username, password string, cram bool) {
	cr, user, passwd, host := parseServer(addr)
	if cr != cram || server != host || user != username || password != passwd {
		t.Errorf("Expecting %v, %v, %v, %v, got %v, %v, %v, %v",
			server, username, password, cram, host, user, passwd, cr)
	}
}

func TestCredentials(t *testing.T) {
	testCredentials(t, "smtp.example.com", "smtp.example.com", "", "", false)
	testCredentials(t, "pepe:lotas@smtp.example.com", "smtp.example.com", "pepe", "lotas", false)
	testCredentials(t, "cram?pepe:lotas@smtp.example.com", "smtp.example.com", "pepe", "lotas", true)
	testCredentials(t, "invalid?pepe:lotas@smtp.example.com", "smtp.example.com", "invalid?pepe", "lotas", false)
	testCredentials(t, "pepe@lotas.com:mayonesa@smtp.example.com", "smtp.example.com", "pepe@lotas.com", "mayonesa", false)
}
