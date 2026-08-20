package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTMDBLookupCachesExpandedMovieSnapshot(t *testing.T) {
	var detailsQuery url.Values
	providerRequested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/3/search/movie":
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Fatalf("authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"results":[{"id":42,"title":"Example Movie"}]}`))
		case "/3/movie/42":
			detailsQuery = r.URL.Query()
			_, _ = w.Write([]byte(`{"id":42,"title":"Example Movie","original_title":"Original Movie","original_language":"tr","production_countries":[{"iso_3166_1":"TR","name":"Turkey"}],"overview":"English overview","release_date":"2024-01-02","vote_average":7.5,"genres":[{"name":"Drama"}],"videos":{"results":[]},"translations":{"translations":[]}}`))
		case "/3/movie/42/watch/providers":
			providerRequested = true
			_, _ = w.Write([]byte(`{"results":{"SA":{"link":"https://example.test/provider","flatrate":[{"provider_id":1,"provider_name":"Example"}]}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewTMDBClient(TMDBConfig{BaseURL: server.URL, BearerToken: "test-token"})
	result, err := client.Lookup(context.Background(), Query{Title: "Example", Type: "movie", Language: "en-US"})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if result.ExternalID != "42" || result.Title != "Example Movie" || result.ReleaseYear != 2024 {
		t.Fatalf("unexpected result: %#v", result)
	}
	for _, resource := range []string{"alternative_titles", "credits", "external_ids", "images", "keywords", "recommendations", "release_dates", "reviews", "similar", "translations", "videos"} {
		if !strings.Contains(detailsQuery.Get("append_to_response"), resource) {
			t.Errorf("append_to_response missing %q: %q", resource, detailsQuery.Get("append_to_response"))
		}
	}
	if !providerRequested {
		t.Fatal("watch-provider endpoint was not requested")
	}
	if !strings.Contains(string(result.RawPayload), `"watch_providers"`) {
		t.Fatalf("raw snapshot does not contain cached providers: %s", result.RawPayload)
	}
	if !containsString(result.Genres, "دراما") || !containsString(result.Genres, "تركي") {
		t.Fatalf("genres should contain normalized genre and country tag: %#v", result.Genres)
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
