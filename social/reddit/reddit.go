package reddit

import (
	"regexp"
)

var (
	commentsRe = regexp.MustCompile("https?://\\w{2,3}\\.reddit\\.\\w{2,3}/r/\\w+/comments/(\\w+)/")
)

func StoryId(url string) string {
	if submatches := commentsRe.FindStringSubmatch(url); submatches != nil {
		return submatches[1]
	}
	return ""
}
