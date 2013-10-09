package facebook

import (
	"testing"
)

func TestLinkStats(t *testing.T) {
	links := []string{"http://www.facebook.com", "www.google.com"}
	stats, err := GetLinksStats(links)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != len(links) {
		t.Fatalf("expecting %d links, got %d instead", len(links), len(stats))
	}
	for _, v := range stats {
		t.Logf("%v", v)
	}
}
