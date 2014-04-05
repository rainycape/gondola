package formatutil

import (
	"fmt"
)

// FormatSize returns the given size in bytes formatted as a
// human readable string. The precision and unit will vary
// depending on the size.
func Size(s uint64) string {
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
