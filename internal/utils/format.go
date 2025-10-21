package utils

import "fmt"

// FormatDuration converts milliseconds to a human-readable duration string
// Very computationally efficient - uses integer division with no allocations
func FormatDuration(durationMs int64) string {
	if durationMs < 0 {
		return "N/A"
	}

	seconds := durationMs / 1000

	// Less than 60 seconds - show seconds
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	// Less than 60 minutes - show minutes and seconds
	if minutes < 60 {
		remainingSeconds := seconds % 60
		if remainingSeconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, remainingSeconds)
	}

	// 60 minutes or more - show hours and minutes
	hours := minutes / 60
	remainingMinutes := minutes % 60
	if remainingMinutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, remainingMinutes)
}
