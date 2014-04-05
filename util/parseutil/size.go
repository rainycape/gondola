package parseutil

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Size parses a size with an optional suffix
// indicating the unit and returns the size in bytes
// Supported suffixes (case insensitive):
//
//  K, KB: kilobytes
//  M, MB: megabytes
//  G, GB: gigabytes
//  T, TB: terabytes
//
// The number before the suffix is parsed as a float,
// so the size might be specified with decimal places
// like e.g. 1.5GB.
func Size(s string) (uint64, error) {
	mult := 1.0
	var n string
	var suffix string
	for ii, v := range s {
		if !unicode.IsDigit(v) && v != '.' {
			n = s[:ii]
			suffix = s[ii:]
			break
		}
	}
	if n == "" && suffix == "" {
		n = s
	}
	switch strings.ToUpper(suffix) {
	case "K", "KB":
		mult = 1024.0
	case "M", "MB":
		mult = 1024.0 * 1024.0
	case "G", "GB":
		mult = 1024.0 * 1024.0 * 1024.0
	case "T", "TB":
		mult = 1024.0 * 1024.0 * 1024.0 * 1024.0
	case "":
		break
	default:
		return 0, fmt.Errorf("invalid suffix %q", suffix)
	}
	val, err := strconv.ParseFloat(n, 64)
	if err != nil {
		return 0, err
	}
	return uint64(val * mult), nil
}
