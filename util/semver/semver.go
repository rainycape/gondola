// Package semver implements a parser and a comparator of
// semantic version numbers.
//
// See http://semver.org for more information.
package semver

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	fieldRe = regexp.MustCompile("^[0-9A-Za-z\\-]+$")
)

// Errors that might be returned from Parse.
var (
	ErrInvalidBuild      = errors.New("build contains invalid characters")
	ErrInvalidPreRelease = errors.New("pre-preleae contains invalid characters")
	ErrMajorNotANumber   = errors.New("major version is not a number")
	ErrMinorNotANumber   = errors.New("minor version is not a number")
	ErrPatchNotANumber   = errors.New("patch version is not a number")
	ErrEmpty             = errors.New("version is empty")
	ErrTooManyFields     = errors.New("version has too many fields (max. is 3)")
)

// A Version represents a semantic version number. Use
// String to convert it to a string and Parse to parse
// a version string into a *Version.
type Version struct {
	Major      int    // Major version number
	Minor      int    // Minor version number
	Patch      int    // Patch version number
	PreRelease string // Pre-release name
	Build      string // Build metadata
}

func (v *Version) String() string {
	var buf bytes.Buffer
	buf.WriteString(strconv.Itoa(v.Major))
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(v.Minor))
	if v.Patch > 0 {
		buf.WriteByte('.')
		buf.WriteString(strconv.Itoa(v.Patch))
	}
	if v.PreRelease != "" {
		buf.WriteByte('-')
		buf.WriteString(v.PreRelease)
	}
	if v.Build != "" {
		buf.WriteByte('+')
		buf.WriteString(v.Build)
	}
	return buf.String()
}

func validate(s string) bool {
	// semver.org
	// A series of dot separated identifiers. Identifiers MUST
	// comprise only ASCII alphanumerics and hyphen [0-9A-Za-z-].
	for _, v := range strings.Split(s, ".") {
		if !fieldRe.MatchString(v) {
			return false
		}
	}
	return true
}

// Parse parses a version string into a *Version.
// The version must follow the rules detailed at
// http://semver.org.
func Parse(version string) (*Version, error) {
	v := &Version{}
	if plus := strings.IndexByte(version, '+'); plus >= 0 {
		v.Build = version[plus+1:]
		if !validate(v.Build) {
			return nil, ErrInvalidBuild
		}
		version = version[:plus]
	}
	if minus := strings.IndexByte(version, '-'); minus >= 0 {
		v.PreRelease = version[minus+1:]
		if !validate(v.PreRelease) {
			return nil, ErrInvalidPreRelease
		}
		version = version[:minus]
	}
	if version == "" {
		return nil, ErrEmpty
	}
	fields := strings.Split(version, ".")
	var err error
	if v.Major, err = strconv.Atoi(fields[0]); err != nil {
		return nil, ErrMajorNotANumber
	}
	if len(fields) > 1 {
		if v.Minor, err = strconv.Atoi(fields[1]); err != nil {
			return nil, ErrMinorNotANumber
		}
		if len(fields) > 2 {
			if v.Patch, err = strconv.Atoi(fields[2]); err != nil {
				return nil, ErrPatchNotANumber
			}
			if len(fields) > 3 {
				return nil, ErrTooManyFields
			}
		}
	}
	return v, nil
}

// Lower is a shorthand for v.Compare(w) < 0.
func (v *Version) Lower(w *Version) bool {
	return v.Compare(w) < 0
}

// Higher is a shorthand for v.Compare(w) > 0.
func (v *Version) Higher(w *Version) bool {
	return v.Compare(w) > 0
}

// Equal returns true iff v and w represent the same
// version number. According to the semver specification,
// build numbers might still differ.
func (v *Version) Equal(w *Version) bool {
	return v.Compare(w) == 0
}

// Compare compares two Version instances. When calling v.Compare(w), the return
// value is as follows:
//
//   < 0: v has a lower version number than w
//  == 0: v and w have equal version number (build numbers might differ)
//   > 0: v has a higher version number than w
func (v *Version) Compare(w *Version) int {
	if v.Major < w.Major {
		return -1
	}
	if v.Major > w.Major {
		return 1
	}
	// Major is equal
	if v.Minor < w.Minor {
		return -1
	}
	if v.Minor > w.Minor {
		return 1
	}
	// Major and minor are equal
	if v.Patch < w.Patch {
		return -1
	}
	if v.Patch > w.Patch {
		return 1
	}
	// Major, minor and patch are equal, compare pre-release.
	// semver.org:
	// When major, minor, and patch are equal, a pre-release
	// version has lower precedence than a normal version.
	if v.PreRelease != "" && w.PreRelease == "" {
		return -1
	}
	if v.PreRelease == "" && w.PreRelease != "" {
		return 1
	}
	if v.PreRelease != "" && w.PreRelease != "" {
		// semver.org:
		// Precedence for two pre-release versions with the same
		// major, minor, and patch version MUST be determined by
		// comparing each dot separated identifier from left to
		// right until a difference is found.
		vFields := strings.Split(v.PreRelease, ".")
		wFields := strings.Split(w.PreRelease, ".")
		vCount := len(vFields)
		wCount := len(wFields)
		end := vCount
		if wCount < end {
			end = wCount
		}
		for ii := 0; ii < end; ii++ {
			iv := vFields[ii]
			iw := wFields[ii]
			vint, verr := strconv.Atoi(iv)
			wint, werr := strconv.Atoi(iw)
			// semver.org
			// Numeric identifiers always have lower precedence
			// than non-numeric identifiers.
			if verr == nil && werr != nil {
				return -1
			}
			if verr != nil && werr == nil {
				return 1
			}
			if verr == nil && werr == nil {
				// Identifiers consisting of only digits are compared
				// numerically.
				if vint < wint {
					return -1
				}
				if vint > wint {
					return 1
				}
			} else {
				// Identifiers with letters or hyphens are compared
				// lexically in ASCII sort order.
				if iv < iw {
					return -1
				}
				if iv > iw {
					return 1
				}
			}
		}
		// semver.org:
		// A larger set of pre-release fields has a higher precedence
		// than a smaller set, if all of the preceding identifiers are
		// equal.
		if vCount < wCount {
			return -1
		}
		if vCount > wCount {
			return 1
		}
	}
	// Everything is equal
	return 0
}
