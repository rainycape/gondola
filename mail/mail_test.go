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

type Validation struct {
	Address    string
	UseNetwork bool
	Valid      bool
}

func TestValidation(t *testing.T) {
	cases := []Validation{
		{"pepe  @gmail.com", true, false},
		{"pepe@lotas@gmail.com", true, false},
		{"pepe", true, false},
		{"pepe@", true, false},
		{"@gmail.com", true, false},
		{"pepe@gmail.com", true, true},
		{"fiam@raichu.rm-fr.net", true, true},
		{"pepe@gmaildoesnotexistwolololhopefullynooneregistersthisdomainandbreaksthistest.com", false, true},
		{"pepe@gmaildoesnotexistwolololhopefullynooneregistersthisdomainandbreaksthistest.com", true, false},
	}
	for _, v := range cases {
		err := Validate(v.Address, v.UseNetwork)
		t.Logf("Validated address %q (net %v), error: %v", v.Address, v.UseNetwork, err)
		valid := err == nil
		if valid != v.Valid {
			e := "valid"
			if valid {
				e = "invalid"
			}
			t.Errorf("Error validating %q (net %v), expecting %s address", v.Address, v.UseNetwork, e)
		}
	}
}
