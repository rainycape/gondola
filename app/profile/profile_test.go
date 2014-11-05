package profile

import "testing"

func TestID(t *testing.T) {
	id := ID()
	t.Logf("id is %v", id)
	if On && id <= 0 {
		t.Fatal("zero id")
	} else if !On && id > 0 {
		t.Fatalf("id is %v, must be <= 0", id)
	}
}
