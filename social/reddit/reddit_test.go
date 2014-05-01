package reddit

import (
	"testing"
)

func TestTitle(t *testing.T) {
	app := &App{}
	story, err := app.Story("10tyhf")
	if err != nil {
		t.Fatalf("error fetching story: %v\n", err)
	}
	title := "I am a multimillionaire AMAA"
	if story.Title != title {
		t.Errorf("invalid story title, expected %v, got %v\n", title, story.Title)
	}
}

func TestUrl(t *testing.T) {
	app := &App{}
	story, err := app.Story("10u32l")
	if err != nil {
		t.Fatalf("error fetching story: %v\n", err)
	}
	url := "http://i.imgur.com/94vHu.png"
	if story.URL != url {
		t.Errorf("invalid story url, expected %v, got %v\n", url, story.URL)
	}
}
