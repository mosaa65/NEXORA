// Command probesearch runs read-only queries against the episode index to verify
// the local enrichment actually produced searchable data.
//
// It talks to Meilisearch directly rather than through the API so it works even
// before the new endpoints are wired, and it never writes anything.
//
// Usage: go run ./cmd/probesearch
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

func main() {
	host := envOr("NEXORA_MEILI_HOST", "http://127.0.0.1:7700")
	client := &http.Client{Timeout: 15 * time.Second}

	// 1. Confirm the index exists and report its shape.
	stats := get(client, host+"/indexes/media_episodes/stats")
	fmt.Printf("=== EPISODE INDEX ===\n%s\n", truncate(stats, 500))

	// 2. Search by an episode title. This is the query the old system could not
	//    answer at all, because no episode documents existed.
	search(client, host, "Ozymandias", "")

	// 3. Filter by work, which is the "episodes of this show" case the separate
	//    index exists for.
	filterByWork(client, host)

	// 4. Search an Arabic work name, which exercises the normalized title.
	search(client, host, "الحلقة", "")
}

func search(client *http.Client, host, query, filter string) {
	body := map[string]any{"q": query, "limit": 3}
	if filter != "" {
		body["filter"] = filter
	}
	payload, _ := json.Marshal(body)

	response, err := client.Post(host+"/indexes/media_episodes/search", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Printf("search %q failed: %v\n", query, err)
		return
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)

	var result struct {
		EstimatedTotalHits int `json:"estimatedTotalHits"`
		Hits               []struct {
			ID            int64  `json:"id"`
			WorkTitleEN   string `json:"work_title_en"`
			SeasonNumber  int    `json:"season_number"`
			EpisodeNumber int    `json:"episode_number"`
			EpisodeTitle  string `json:"episode_title_en"`
			HasLocalFile  bool   `json:"has_local_file"`
			EnrichedFrom  string `json:"enriched_from"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		fmt.Printf("search %q: unreadable response: %s\n", query, truncate(string(raw), 200))
		return
	}

	fmt.Printf("=== SEARCH q=%q hits=%d ===\n", query, result.EstimatedTotalHits)
	for _, hit := range result.Hits {
		fmt.Printf("  S%02dE%02d  %-34s  %s  file=%v via=%s\n",
			hit.SeasonNumber, hit.EpisodeNumber, truncate(hit.EpisodeTitle, 34),
			truncate(hit.WorkTitleEN, 24), hit.HasLocalFile, hit.EnrichedFrom)
	}
	fmt.Println()
}

func filterByWork(client *http.Client, host string) {
	body := map[string]any{
		"q":      "",
		"limit":  3,
		"filter": "work_id = 234",
		"sort":   []string{"season_number:asc", "episode_number:asc"},
	}
	payload, _ := json.Marshal(body)

	response, err := client.Post(host+"/indexes/media_episodes/search", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Printf("work filter failed: %v\n", err)
		return
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)

	var result struct {
		EstimatedTotalHits int `json:"estimatedTotalHits"`
		Hits               []struct {
			SeasonNumber  int    `json:"season_number"`
			EpisodeNumber int    `json:"episode_number"`
			EpisodeTitle  string `json:"episode_title_en"`
			AirDate       string `json:"air_date"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		fmt.Printf("work filter: %s\n", truncate(string(raw), 200))
		return
	}

	fmt.Printf("=== FILTER work_id=234 hits=%d ===\n", result.EstimatedTotalHits)
	for _, hit := range result.Hits {
		fmt.Printf("  S%02dE%02d  %-34s  air=%s\n",
			hit.SeasonNumber, hit.EpisodeNumber, truncate(hit.EpisodeTitle, 34), hit.AirDate)
	}
	fmt.Println()
}

func get(client *http.Client, url string) string {
	response, err := client.Get(url)
	if err != nil {
		return "unreachable: " + err.Error()
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	return string(raw)
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
