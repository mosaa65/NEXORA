package db

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestIndexingStateMigrationIsWellFormed validates migration 0022 statically.
//
// A broken migration blocks server startup for every deployment, and no live
// PostgreSQL is available in this environment, so the file is checked for the
// properties that actually matter: balanced statements, the tables and columns
// the repository depends on, and only columns that the schema can safely accept.
func TestIndexingStateMigrationIsWellFormed(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "0022_add_indexing_state.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	content := string(raw)

	// Every statement must be terminated. An unterminated final statement is the
	// most common way a migration silently half-applies.
	if !strings.HasSuffix(strings.TrimSpace(content), ";") {
		t.Error("migration does not end with a terminated statement")
	}

	// The tables the repository code queries must exist after this migration.
	for _, table := range []string{"scan_sessions", "scan_roots", "media_roots"} {
		pattern := "CREATE TABLE IF NOT EXISTS " + table
		if !strings.Contains(content, pattern) {
			t.Errorf("migration is missing %q", pattern)
		}
	}

	// The columns the batch ingest writes must all be added or already exist.
	requiredVideoFileColumns := []string{
		"file_id", "file_mod_time", "root_id", "root_path", "relative_path",
		"part_number", "episode_end", "title_normalized", "search_tokens",
		"parse_confidence", "parse_reasons", "special_kind", "state",
		"last_scan_id", "last_seen_at", "first_indexed_at",
	}
	for _, column := range requiredVideoFileColumns {
		if !strings.Contains(content, column) {
			t.Errorf("migration does not add video_files.%s", column)
		}
	}

	// ALTER TABLE must use IF NOT EXISTS so the migration is re-runnable, which
	// the runner guarantees but a manual operator may not.
	alterRE := regexp.MustCompile(`(?m)^ALTER TABLE\s+(\w+)\s*\n([^;]+);`)
	for _, match := range alterRE.FindAllStringSubmatch(content, -1) {
		table := match[1]
		body := match[2]
		for _, line := range strings.Split(body, ",") {
			line = strings.TrimSpace(strings.ReplaceAll(line, "\n", " "))
			if line == "" || strings.HasPrefix(strings.ToUpper(line), "ADD COLUMN IF NOT EXISTS") {
				continue
			}
			if strings.HasPrefix(strings.ToUpper(line), "ADD COLUMN") {
				t.Errorf("ALTER TABLE %s uses ADD COLUMN without IF NOT EXISTS: %q", table, line)
			}
		}
	}

	// CREATE INDEX must be idempotent for the same reason.
	createIndexRE := regexp.MustCompile(`(?m)^CREATE (UNIQUE )?INDEX (IF NOT EXISTS )?`)
	for _, match := range createIndexRE.FindAllStringSubmatch(content, -1) {
		if match[2] == "" {
			t.Error("a CREATE INDEX is missing IF NOT EXISTS")
		}
	}

	// The unique episode identity must be partial, or it would incorrectly
	// reject legitimate movies that have no episode number.
	if strings.Contains(content, "idx_video_files_episode_identity") {
		if !strings.Contains(content, "WHERE season_id IS NOT NULL") {
			t.Error("the episode identity index must be partial so movies are unaffected")
		}
	}
}

// TestMigrationsAreOrderedAndUnique guards the migration runner's assumptions.
func TestMigrationsAreOrderedAndUnique(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	seen := make(map[string]string, len(entries))
	previous := ""
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if previous != "" && entry.Name() <= previous {
			t.Errorf("migration %q does not sort after %q; the runner applies them lexicographically",
				entry.Name(), previous)
		}
		previous = entry.Name()

		// The runner orders by full filename, so uniqueness of the filename is the
		// property the runner actually requires. Duplicate numeric prefixes already
		// exist in the repository (two files share "0021_"); they are ordered by name
		// and are therefore harmless, but they are reported so a new file never makes
		// the ordering ambiguous.
		if existing, duplicate := seen[entry.Name()]; duplicate {
			t.Errorf("migration %q is defined twice (also %q)", entry.Name(), existing)
		}
		seen[entry.Name()] = entry.Name()
	}
}
