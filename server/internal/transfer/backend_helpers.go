package transfer

import (
	"os"
	"path/filepath"
	"strings"
)

func statSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func splitRemoteDir(remotePath string) []string {
	var parts []string
	for _, part := range strings.Split(strings.ReplaceAll(remotePath, "\\", "/"), "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func resolveTool(devicePath, toolName string) string {
	if devicePath != "" {
		for _, suffix := range []string{"", ".exe"} {
			candidate := filepath.Join(devicePath, toolName+suffix)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return toolName
}

func pathLeaf(input string) string {
	trimmed := strings.Trim(input, "/")
	if trimmed == "" || trimmed == "." {
		return "Documents"
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}

// leafName returns the final path segment of a path on any platform. Unlike
// pathLeaf it also splits on backslashes so Windows temp/source paths produce
// the real file name (used by the Android MTP backend for target naming).
func leafName(input string) string {
	normalized := strings.ReplaceAll(input, "\\", "/")
	normalized = strings.Trim(normalized, "/")
	if normalized == "" {
		return ""
	}
	parts := strings.Split(normalized, "/")
	return parts[len(parts)-1]
}
