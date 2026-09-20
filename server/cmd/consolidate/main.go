// Command consolidate repairs duplicate works in the catalogue.
//
// It exists because the pre-ADR-010 ingest created one work per filename, so the
// same show appears several times and container folders became works. This tool
// finds those rows, and — only when told to — folds them together safely.
//
// It is dry-run by default. A merge rewrites catalogue rows, so it must never be
// the default behaviour of a command an operator runs to look at the problem.
//
// Usage:
//
//	go run ./cmd/consolidate                 # report only
//	go run ./cmd/consolidate -apply          # merge the safe groups
//	go run ./cmd/consolidate -reresolve      # repair container-titled rows
//	go run ./cmd/consolidate -apply -reresolve -project
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
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

func main() {
	apply := flag.Bool("apply", false, "execute the merge instead of only reporting it")
	reresolve := flag.Bool("reresolve", false, "repair rows whose title is a container folder")
	project := flag.Bool("project", false, "rebuild both search indexes afterwards")
	inspect := flag.Bool("inspect", false, "print every group in detail, not only the totals")
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

	// The repair reuses the scanner's own path parser, so a recovered title and
	// a freshly ingested title cannot differ for the same folder layout.
	db.SetReresolveParser(func(path string) (string, string, string, string) {
		parsed := scanner.ParseFilePath(path)
		// ParsedName carries isEpisode rather than a media type, so the type is
		// derived from the same signals the classifier uses: an episodic file is a
		// series, anything else is a film. The re-resolution only needs this to keep
		// the row's type consistent with its recovered title.
		mediaType := "movie"
		if parsed.IsEpisode {
			mediaType = "series"
		}
		return parsed.Title, parsed.TitleAR, mediaType, parsed.CategorySlug
	})

	// Repair container-titled rows BEFORE looking for duplicates.
	//
	// These rows are textually identical to each other (several rows all named
	// "أعمال"), so a duplicate scan run first can only see them as one repeated
	// title. They are not duplicates: each lost its own title to the same
	// container folder. Re-resolving first gives each row its real title, after
	// which any genuine duplicates become visible.
	if *reresolve {
		fmt.Println()
		stats, err := repository.ReresolveContainerWorks(ctx)
		if err != nil {
			logger.Error("re-resolve failed", slog.Any("error", err))
		} else {
			fmt.Println("=== CONTAINER FOLDER REPAIR ===")
			fmt.Printf("scanned  : %d\n", stats.Scanned)
			fmt.Printf("repaired : %d\n", stats.Resolved)
			fmt.Printf("skipped  : %d\n", stats.Skipped)
			for _, detail := range stats.Details {
				fmt.Printf("  repaired  work %-5d  %q -> %q\n", detail.WorkID, detail.OldTitle, detail.NewTitle)
			}
			for _, skipped := range stats.SkippedDetails {
				fmt.Printf("  skipped   work %-5d  %q  reason: %s\n", skipped.WorkID, skipped.OldTitle, skipped.FromPath)
			}
		}
	}

	groups, err := repository.FindDuplicateGroups(ctx)
	if err != nil {
		logger.Error("duplicate scan failed", slog.Any("error", err))
		os.Exit(1)
	}

	byKind := map[db.DuplicateKind]int{}
	wouldMerge := 0
	for _, group := range groups {
		byKind[group.Kind]++
		if group.Safe {
			wouldMerge += len(group.Members)
		}
	}

	fmt.Println()
	fmt.Println("=== DUPLICATE WORK REPORT ===")
	fmt.Printf("groups found            : %d\n", len(groups))
	fmt.Printf("  same title + type     : %d  (safe to merge)\n", byKind[db.KindSameTitle])
	fmt.Printf("  alias-linked          : %d  (safe to merge)\n", byKind[db.KindAliasLinked])
	fmt.Printf("  container folder      : %d  (needs re-resolve, not merge)\n", byKind[db.KindContainer])
	fmt.Printf("rows that would be folded: %d\n", wouldMerge)

	if *inspect {
		fmt.Println()
		for _, group := range groups {
			fmt.Printf("[%s] canonical=%d  reason=%s\n", group.Kind, group.CanonicalID, group.Reason)
			for _, member := range group.Members {
				fmt.Printf("    fold id=%-5d %-38q type=%-8s files=%d\n",
					member.ID, member.TitleEN, member.MediaType, member.FileCount)
			}
		}
	}

	if !*apply && !*reresolve && !*project {
		fmt.Println()
		fmt.Println("no changes made (dry run). re-run with -apply and/or -reresolve to execute.")
		return
	}

	// Fold the provably-identical rows. Container rows are excluded from this
	// set, so a merge can never fold unrelated films together.
	if *apply {
		fmt.Println()
		stats, err := repository.MergeDuplicateGroups(ctx, groups)
		if err != nil {
			logger.Error("merge failed", slog.Any("error", err))
		} else {
			fmt.Println("=== MERGE RESULT ===")
			fmt.Printf("groups merged     : %d\n", stats.GroupsMerged)
			fmt.Printf("works folded      : %d\n", stats.WorksMerged)
			fmt.Printf("files repointed   : %d\n", stats.FilesRepointed)
			fmt.Printf("seasons moved     : %d\n", stats.SeasonsMoved)
			fmt.Printf("episodes moved    : %d\n", stats.EpisodesMoved)
			fmt.Printf("aliases preserved : %d\n", stats.AliasesKept)
			fmt.Printf("skipped           : %d\n", stats.Skipped)
		}
	}

	// 3. Rebuild both indexes, because the merges changed which works exist.
	if *project {
		client := search.NewClient(search.Config{
			Host: cfg.MeiliHost, APIKey: cfg.MeiliAPIKey, Index: cfg.MeiliIndex,
		})
		fmt.Println()
		fmt.Println("=== INDEX REBUILD ===")
		workProjector := search.NewProjector(client, repository, search.DefaultProjectionPageSize, logger)
		workResult, err := workProjector.Rebuild(ctx, true)
		if err != nil {
			logger.Warn("work index rebuild incomplete", slog.Any("error", err))
		}
		fmt.Printf("works indexed : %d (pages %d, pruned %d)\n",
			workResult.Documents, workResult.Pages, workResult.Pruned)

		episodeProjector := search.NewEpisodeProjector(client, repository, search.DefaultProjectionPageSize, logger)
		episodeResult, err := episodeProjector.Rebuild(ctx, true)
		if err != nil {
			logger.Warn("episode index rebuild incomplete", slog.Any("error", err))
		}
		fmt.Printf("episodes indexed: %d (pages %d)\n", episodeResult.Documents, episodeResult.Pages)
	}
}
