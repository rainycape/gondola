package util

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseSize parses a size with an optional suffix
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
func ParseSize(s string) (uint64, error) {
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

// FormatSize returns the given size in bytes formatted as a
// human readable string. The precision and unit will vary
// depending on the size.
func FormatSize(s uint64) string {
	if s < 1024 {
		return fmt.Sprintf("%d bytes", s)
	}
	if s < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(s)/1024)
	}
	if s < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(s)/(1024*1024))
	}
	if s < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.3f GB", float64(s)/(1024*1024*1024))
	}
	return fmt.Sprintf("%.4f TB", float64(s)/(1024*1024*1024*1024))
}
