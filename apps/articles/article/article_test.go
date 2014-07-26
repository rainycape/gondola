package article

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestArticle(t *testing.T) {
	p := filepath.Join("testdata", "test.md")
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	article, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(article.Properties) > 0 {
		t.Fatalf("article should have no unknown properties, it has %v", article.Properties)
	}
	var buf bytes.Buffer
	if _, err := article.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	article2, err := New(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(article, article2) {
		t.Fatalf("writing and loading article changed it\nfrom %+v\n  to %+v", article, article2)
	}
}
