package transfer

import "strings"

// Path sanitization helpers used by every device backend (iOS, Android,
// Storage). They prevent path traversal (../), absolute-escape and invalid
// separators before a remote path is passed to any external tool.

// safePathComponent cleans a single path segment, replacing characters that are
// invalid on Windows / remote filesystems with an underscore. An empty segment
// resolves to "Unknown".
func safePathComponent(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "Unknown"
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, input)
}

// safeRelativePath normalises a user-supplied relative path for use as a remote
// destination. It collapses backslashes, strips leading separators/"./" and
// drops segments that are empty, "." or ".." (along with "Unknown").
func safeRelativePath(input string) string {
	input = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	if input == "" {
		return ""
	}
	input = strings.TrimPrefix(input, "/")
	input = strings.TrimPrefix(input, "./")
	parts := make([]string, 0)
	for _, part := range strings.Split(input, "/") {
		clean := safePathComponent(part)
		if clean == "" || clean == "." || clean == ".." || clean == "Unknown" {
			continue
		}
		parts = append(parts, clean)
	}
	return strings.Join(parts, "/")
}

// splitRelativePath is safeRelativePath followed by a "/" split. Returns nil
// for an empty/normalised-empty path.
func splitRelativePath(path string) []string {
	path = safeRelativePath(path)
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
