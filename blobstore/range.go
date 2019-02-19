package blobstore

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Range represents given byte range in a file. To obtain
// a Range from an incoming request, use ParseRange().
type Range struct {
	Start *int64
	End   *int64
}

func (r *Range) empty() bool {
	return r == nil || (r.Start == nil && r.End == nil)
}

// IsValid returns true iff r is not nil and either Start or End
// are non-zero.
func (r *Range) IsValid() bool {
	return !r.empty()
}

// Range returns the Range's Start and End, which
// might be nil.
func (r *Range) Range() (*int64, *int64) {
	return r.Start, r.End
}

// StatusCode returns the HTTP response status code which
// should be using when serving a response with this range.
func (r *Range) StatusCode() int {
	if r.empty() {
		return http.StatusOK
	}
	return http.StatusPartialContent
}

// Size returns the number of bytes enclosed by this range. If the
// range is empty, it returns 0.
func (r *Range) Size(total uint64) uint64 {
	if r.empty() {
		return 0
	}
	if r.Start == nil {
		return uint64(*r.End)
	}
	if *r.Start < 0 {
		return uint64(-*r.Start)
	}
	if r.End == nil {
		return total + uint64(*r.Start)
	}
	// Range 0-9 should serve 10 bytes
	return uint64(*r.End - *r.Start + 1)
}

// Set sets the corresponding headers for this Range on the given
// http.ResponseWriter.
func (r *Range) Set(w http.ResponseWriter, total uint64) {
	if !r.empty() {
		header := w.Header()
		if s := r.Size(total); s != 0 {
			header.Set("Content-Length", strconv.FormatUint(s, 10))
		}
		header.Set("Content-Range", r.responseString())
	}
}

func (r *Range) String() string {
	return r.makeString("=")
}

func (r *Range) makeString(sep string) string {
	if r.IsValid() {
		if r.Start != nil && r.End != nil {
			return fmt.Sprintf("bytes%s%d-%d", sep, *r.Start, *r.End)
		}
		if r.Start != nil {
			if *r.Start < 0 {
				return fmt.Sprintf("bytes%s%d", sep, *r.Start)
			}
			return fmt.Sprintf("bytes%s%d-", sep, *r.Start)
		}
		if r.End != nil {
			return fmt.Sprintf("bytes%s-%d", sep, *r.End)
		}
	}
	return ""
}

func (r *Range) responseString() string {
	return r.makeString(" ") + "/*"
}

// ParseRange returns a *Range from the given *http.Request if it
// has a well formed Range header, otherwise returns nil.
func ParseRange(r *http.Request) *Range {
	if r == nil {
		return nil
	}
	const prefix = "bytes="
	rng := strings.TrimSpace(r.Header.Get("Range"))
	if strings.HasPrefix(rng, prefix) {
		rng = rng[len(prefix):]
		p := strings.Split(rng, "-")
		if len(p) == 2 {
			var start, end *int64
			if p[0] != "" {
				s, err := strconv.ParseInt(p[0], 10, 64)
				if err != nil {
					return nil
				}
				start = &s
			}
			if p[1] != "" {
				e, err := strconv.ParseInt(p[1], 10, 64)
				if err != nil {
					return nil
				}
				end = &e
			}
			if start != nil && end != nil && *start > *end {
				return nil
			}
			// Tail range request
			if start == nil && end != nil {
				start = end
				*start = -*start
				end = nil
			}
			return &Range{Start: start, End: end}
		}
	}
	return nil
}
