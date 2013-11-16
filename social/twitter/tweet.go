package twitter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	MAX_TWEET_LENGTH    = 140
	URL_CONSUMED_LENGTH = 23
	ellipsis            = "\u2026"
)

var (
	linkRe = regexp.MustCompile("i?(\\s|^)https?://\\S+")
)

type Tweet struct {
	Id   string `json:"id_str"`
	Text string `json:"text"`
}

// TweetOptions contains some optional settings which may be
// specified when sending a tweet.
type TweetOptions struct {
	// If true, text will get truncated if it's longer than
	// the maximum tweet length (instead of returning an error).
	Truncate bool
	// Only has effect if Truncate is true. Rather than truncating
	// at 140 characters, it will truncate at the last whitespace.
	TruncateOnWhitespace bool
	// The ID of an existing status that the update is in reply to.
	// Note that Twitter ignores this unless the referenced id's
	// author is @mentioned in the tweet text.
	InReplyTo string
	// The latitude of the location this tweet refers to.
	Latitude float64
	// The longitude of the location this tweet refers to.
	Longitude float64
	// This parameter only has effect when Latitude and
	// Longitude are set. If this parameter is false, the
	// tweet will contain a pin with the exact location. Set
	// it to true to hide the pin.
	HideCoordinates bool
	// A place retrieved from geo/reverse_geocode
	PlaceId string
}

// Update sends a tweet with the given text, using the provided app and token.
// Options are optional, you might pass nil.
func (app *App) Update(text string, token *Token, opts *TweetOptions) (*Tweet, error) {
	count := countCharacters(text)
	if count > MAX_TWEET_LENGTH {
		if opts == nil || !opts.Truncate {
			return nil, fmt.Errorf("tweet is too long (%d effective characters)", count)
		}
		text = truncateText(text, count, opts)
	}
	values := map[string]string{"status": text}
	if opts != nil {
		if opts.InReplyTo != "" {
			values["in_reply_to_status_id"] = opts.InReplyTo
		}
		if opts.Latitude != 0 || opts.Longitude != 0 {
			values["lat"] = strconv.FormatFloat(opts.Latitude, 'f', -1, 64)
			values["long"] = strconv.FormatFloat(opts.Longitude, 'f', -1, 64)
			if opts.HideCoordinates {
				values["display_coordinates"] = "false"
			} else {
				values["display_coordinates"] = "true"
			}
		}
		if opts.PlaceId != "" {
			values["place_id"] = opts.PlaceId
		}
	}
	var tw Tweet
	err := sendReq(app, token, "POST", statusPath, values, &tw)
	if err != nil {
		return nil, err
	}
	return &tw, nil
}

// CharacterCount returns the effective number of characters for a
// tweet. Keep in mind that URLs always take URL_CONSUMED_LENGTH,
// so it might differ from the exact number of characters in the tweet.
func countCharacters(text string) int {
	// replace urls with an empty string, count the remaning and add
	// URL_CONSUMED_LENGTH * number_of_urls.
	count := 0
	extra := 0
	replaced := linkRe.ReplaceAllStringFunc(text, func(s string) string {
		count++
		if s != "" && s[0] == ' ' {
			extra++
		}
		return ""
	})
	return len(replaced) + (count * URL_CONSUMED_LENGTH) + extra
}

func truncateText(text string, count int, opts *TweetOptions) string {
	// Must remove 1 more character for the ellipsis
	toRemove := (count - MAX_TWEET_LENGTH) + 1
	if toRemove <= 0 {
		return text
	}
	// Try to keep all the urls in the post, truncate from
	// "non-special" text (we will probably do the same for
	// @mentions in the near future).
	special := -1
	links := linkRe.FindAllStringIndex(text, -1)
	if len(links) > 0 {
		special = len(links) - 1
	}
	removed := 0
	end := len(text) - 1
	var suffixes []string
	for cur := end; cur >= 0; cur-- {
		if special >= 0 && links[special][1]-1 == cur {
			// Reached a special block
			specialText := text[links[special][0]:links[special][1]]
			suffixes = append(suffixes, specialText)
			cur = links[special][0]
			end = cur - 1
			if specialText[0] == ' ' {
				end--
			}
			special--
		} else {
			removed++
			end--
			if toRemove == removed {
				break
			}
		}
	}
	for toRemove > removed && len(suffixes) > 0 {
		last := suffixes[len(suffixes)-1]
		suffixes = suffixes[:len(suffixes)-1]
		removed += URL_CONSUMED_LENGTH
		if last[0] == ' ' {
			removed += 1
		}
	}
	if end >= 0 {
		return text[:end] + ellipsis + strings.Join(suffixes, "")
	}
	if len(suffixes) > 0 && suffixes[0][0] == ' ' {
		suffixes[0] = suffixes[0][1:]
	}
	return strings.Join(suffixes, "")
}
