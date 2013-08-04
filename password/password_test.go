package password

import (
	"testing"
)

func TestPassword(t *testing.T) {
	pw := "gondola"
	for _, v := range []Hash{SHA1, SHA224, SHA256, SHA384, SHA512} {
		p := NewHashed(pw, v)
		t.Logf("Password %q was encoded using %s as %q", pw, v.Name(), p.String())
		if err := p.Check(pw); err != nil {
			t.Errorf("Error verifying password %q using %s: %s", pw, v.Name(), err)
		}
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
		if Password(v).IsValid() {
			t.Errorf("Invalid password %q parsed as valid", v)
		}
	}
}
