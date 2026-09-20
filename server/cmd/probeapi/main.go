// Command probeapi inspects the real API responses the work-details screen will
// consume, so the UI is built against the actual payload shape.
//
// It is read-only and talks to a running server.
//
// Usage: go run ./cmd/probeapi
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const apiBase = "http://127.0.0.1:8080"

func main() {
	client := &http.Client{Timeout: 20 * time.Second}

	// 1. What a work-details response actually contains for a show with many
	//    seasons and enriched episodes.
	show(client, "/api/media/160", "One Piece (23 seasons, enriched)")

	// 2. The episode index, filtered to one work. This is the query the new
	//    details screen uses instead of walking the local season tree.
	search(client, "/api/episodes/search?work=160&limit=5&season=1")
	search(client, "/api/episodes/search?work=160&local=false&limit=5")

	// 3. The metadata snapshot route the current page uses.
	show(client, "/api/media/160/metadata/seasons?locale=ar-SA", "season snapshots")
}

func show(client *http.Client, path, label string) {
	response, err := client.Get(apiBase + path)
	if err != nil {
		fmt.Printf("[%s] request failed: %v\n", label, err)
		return
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))

	fmt.Printf("\n=== %s ===\nGET %s -> %d\n", label, path, response.StatusCode)
	if response.StatusCode >= 300 {
		fmt.Println(truncate(string(raw), 300))
		return
	}
	fmt.Println(summarize(raw))
}

func search(client *http.Client, path string) {
	show(client, path, "episode search")
}

// summarize reports the top-level keys and a few counts rather than dumping the
// whole payload, which is what makes the shape readable at a glance.
func summarize(raw []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return truncate(string(raw), 300)
	}

	var out bytes.Buffer
	for _, key := range []string{"id", "title_ar", "title_en", "type", "release_year",
		"poster_path", "file_count", "season_count", "estimatedTotalHits", "total"} {
		if value, exists := payload[key]; exists {
			fmt.Fprintf(&out, "  %-20s = %v\n", key, value)
		}
	}

	for _, key := range []string{"seasons", "files", "hits", "items"} {
		if list, ok := payload[key].([]any); ok {
			fmt.Fprintf(&out, "  %-20s = %d entries\n", key, len(list))
			if len(list) > 0 {
				if first, ok := list[0].(map[string]any); ok {
					keys := make([]string, 0, len(first))
					for k := range first {
						keys = append(keys, k)
					}
					fmt.Fprintf(&out, "    first entry keys: %s\n", strings.Join(keys, ", "))
				}
			}
		}
	}
	return out.String()
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

var _ = os.Getenv
