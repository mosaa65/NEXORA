package transfer

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Compute the overall transfer timeout for a payload of the given size.
// A 90s floor is used; for larger payloads the estimate is
// (size / 4MB) seconds, and the value is capped at 90 minutes.
func transferTimeout(size int64) time.Duration {
	minTimeout := 90 * time.Second
	if size <= 0 {
		return minTimeout
	}
	estimated := time.Duration(size/(4*1024*1024)) * time.Second
	if estimated < minTimeout {
		return minTimeout
	}
	if estimated > 90*time.Minute {
		return 90 * time.Minute
	}
	return estimated + 45*time.Second
}

// formatBytes renders a byte count in a human-readable form (B/KB/MB/GB/TB/PB).
func formatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	for _, unit := range units {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", value/1024)
}

var sizeRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*([kmgt]?i?b|bytes?)?$`)

// parseHumanOrNumericSize parses a human or numeric size string (e.g.
// "734003200", "700MB", "1.2 GB") into a raw byte count.
func parseHumanOrNumericSize(value string) (int64, error) {
	raw := strings.TrimSpace(strings.ToLower(value))
	raw = strings.ReplaceAll(raw, ",", "")
	if raw == "" {
		return 0, errors.New("empty size")
	}
	match := sizeRe.FindStringSubmatch(raw)
	if len(match) != 3 {
		return 0, fmt.Errorf("invalid size: %s", value)
	}
	number, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, err
	}
	unit := strings.TrimSpace(match[2])
	multiplier := float64(1)
	switch unit {
	case "kb", "kib":
		multiplier = 1024
	case "mb", "mib":
		multiplier = 1024 * 1024
	case "gb", "gib":
		multiplier = 1024 * 1024 * 1024
	case "tb", "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	}
	return int64(number * multiplier), nil
}
