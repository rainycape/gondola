package password

import (
	"testing"
)

func TestPassword(t *testing.T) {
	pw := "whatever"
	p := New(pw)
	t.Logf("Password %q was encoded as %q", pw, p.String())
	if err := p.Check(pw); err != nil {
		t.Errorf("Error verifying password %q: %s", pw, err)
	}
}

func TestInvalidPasswords(t *testing.T) {
	invalid := []string{
		"",
		"foo",
		"pepe:lotas",
		"pepe:lotas:foo",
		"pepe:lotas:a2e4150de3aec65b826e6105392058a42cf3c63cc2ab859a68962603ae8a0588",
	}
	for _, v := range invalid {
		if Password(v).Valid() {
			t.Errorf("Invalid password %q parsed as valid", v)
		}
	}
}
