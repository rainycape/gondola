package reddit

import (
	"testing"
)

func TestTitle(t *testing.T) {
	story, err := FetchStory("10tyhf")
	if err != nil {
		t.Errorf("Error fetching story: %v\n", err)
		return
	}
	title := "I am a multimillionaire AMAA"
	if story.Title != title {
		t.Errorf("Invalid story title, expected %v, got %v\n", title, story.Title)
	}
}

func TestUrl(t *testing.T) {
	story, err := FetchStory("10u32l")
	if err != nil {
		t.Errorf("Error fetching story: %v\n", err)
		return
	}
	url := "http://i.imgur.com/94vHu.png"
	if story.URL != url {
		t.Errorf("Invalid story url, expected %v, got %v\n", url, story.URL)
	}
}
