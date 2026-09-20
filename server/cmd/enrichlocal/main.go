// Command enrichlocal fills seasons and episodes from the TMDB snapshots that
// are already stored in this database, with zero external requests.
//
// It exists as a runnable tool so the enrichment can be measured on a real
// library before it is wired to an endpoint, and so an operator can re-run it
// after importing new provider snapshots.
//
// Usage:
//
//	go run ./cmd/enrichlocal              # every work with a stored snapshot
//	go run ./cmd/enrichlocal -work 234    # one work
//	go run ./cmd/enrichlocal -limit 5     # a bounded trial run
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"nexora/server/internal/config"
	"nexora/server/internal/db"
	"nexora/server/internal/search"
)

func main() {
	workID := flag.Int64("work", 0, "enrich one work id only (0 = all works with snapshots)")
	limit := flag.Int("limit", 0, "maximum number of works to enrich (0 = no limit)")
	linkFiles := flag.Bool("link", true, "attach video files to their episode row")
	projectIndex := flag.Bool("project", true, "rebuild the episode search index afterwards")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cfg := config.Load()
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		logger.Error("database unreachable", slog.Any("error", err))
		os.Exit(1)
	}

	repository := db.NewRepository(sqlDB)

	before := countRows(ctx, sqlDB, "SELECT COUNT(*) FROM episodes")

	started := time.Now()
	results, err := repository.EnrichFromLocalSnapshots(ctx, *workID, *limit)
	if err != nil {
		logger.Error("enrichment failed", slog.Any("error", err))
		os.Exit(1)
	}
	elapsed := time.Since(started)

	var seasonsCreated, episodesCreated, episodesEnriched int
	for _, result := range results {
		seasonsCreated += result.SeasonsCreated
		episodesCreated += result.EpisodesCreated
		episodesEnriched += result.EpisodesEnriched
	}

	linked := 0
	if *linkFiles {
		linked, err = repository.LinkOrphanEpisodes(ctx)
		if err != nil {
			logger.Warn("could not link files to episodes", slog.Any("error", err))
		}
	}

	// Project the enriched episodes into their own search index, so the new
	// rows are findable and the inside-a-work search has data to filter.
	projection := search.ProjectionResult{}
	if *projectIndex {
		episodeProjector := search.NewEpisodeProjector(
			search.NewClient(search.Config{Host: cfg.MeiliHost, APIKey: cfg.MeiliAPIKey,
				Index: cfg.MeiliIndex}),
			repository, search.DefaultProjectionPageSize, logger)
		projection, err = episodeProjector.Rebuild(ctx, true)
		if err != nil {
			logger.Warn("episode search projection failed", slog.Any("error", err))
		}
	}

	after := countRows(ctx, sqlDB, "SELECT COUNT(*) FROM episodes")

	fmt.Println()
	fmt.Println("=== LOCAL ENRICHMENT RESULT (zero external requests) ===")
	fmt.Printf("works processed        : %d\n", len(results))
	fmt.Printf("seasons created        : %d\n", seasonsCreated)
	fmt.Printf("episodes created       : %d\n", episodesCreated)
	fmt.Printf("episodes enriched      : %d\n", episodesEnriched)
	fmt.Printf("files linked to eps    : %d\n", linked)
	fmt.Printf("episodes before        : %d\n", before)
	fmt.Printf("episodes after         : %d\n", after)
	if *projectIndex {
		fmt.Printf("indexed for search     : %d (pages %d)\n", projection.Documents, projection.Pages)
	}
	fmt.Printf("wall clock             : %s\n", elapsed.Round(time.Millisecond))
	if after > 0 {
		fmt.Printf("per episode            : %s\n", (elapsed / time.Duration(after)).Round(time.Microsecond))
	}
}

// countRows runs a single-value count query, returning 0 on any failure so the
// report degrades rather than aborting.
func countRows(ctx context.Context, database *sql.DB, query string) int64 {
	var value int64
	if err := database.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return 0
	}
	return value
}
