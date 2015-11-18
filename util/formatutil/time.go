package formatutil

import (
	"time"

	"gnd.la/i18n"
)

// DateStyle represents a style for formatting
// a date. All the examples use January 2nd, 2006.
// A _ before a number represents a space that might
// be replaced by a digit if the following number has
// two digits.
type DateStyle int

const (
	// 02/01/06
	DateStyleShort DateStyle = iota - 1
	// Jan 2, 2006
	DateStyleMedium
	// January 2, 2006
	DateStyleLong
)

const (
	day = 24 * time.Hour
)

// extractDate returns a time.Time with the time part truncated,
// leaving the date part as is.
func extractDate(t time.Time) time.Time {
	return t.Add(-12 * time.Hour).Round(day)
}

func Time(lang i18n.Languager) string {
	return ""
}

func Duration(lang i18n.Languager, d time.Duration) string {
	return ""
}

func TimeSince(lang i18n.Languager, t time.Time) string {
	return Duration(lang, time.Since(t))
}

// Date formats the date part of the given time.Time according to
// the given DateStyle.
func Date(lang i18n.Languager, t time.Time, style DateStyle) string {
	var layout string
	switch style {
	case DateStyleShort:
		/// SHORT DATE FORMAT
		layout = i18n.Tc(lang, "formatutil", "02/01/2006")
	default:
		fallthrough
	case DateStyleMedium:
		/// MEDIUM DATE FORMAT
		layout = i18n.Tc(lang, "formatutil", "Jan 2, 2006")
	case DateStyleLong:
		/// LONG DATE FORMAT
		layout = i18n.Tc(lang, "formatutil", "January 2, 2006")
	}
	return t.Format(layout)
}

// RelativeDate works like Date, but uses relative names when possible. e.g.
// in English "yesterday", "today" and "tomorrow" are the available relative
// names, but other languages might have more or none of them.
func RelativeDate(lang i18n.Languager, t time.Time, style DateStyle) string {
	today := time.Now().Add(-12 * time.Hour).Round(day)
	dateDay := t.Add(-12 * time.Hour).Round(day)

	if today.Equal(dateDay) {
		/// TODAY
		return i18n.Tc(lang, "formatutil", "today")
	}

	if today.AddDate(0, 0, -1).Equal(dateDay) {
		/// YESTERDAY
		return i18n.Tc(lang, "formatutil", "yesterday")
	}

	return Date(lang, t, style)
}
