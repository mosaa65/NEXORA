package app

import (
	"context"
	"log/slog"

	"nexora/server/internal/config"
	"nexora/server/internal/db"
	"nexora/server/internal/scanner"
)

// logInterruptedScans reports scans that never finished. A session left in
// 'running' means the previous process stopped mid-scan, so the catalogue may
// need a reconciliation pass rather than being assumed consistent.
func logInterruptedScans(ctx context.Context, repository *db.Repository) {
	sessions, err := repository.InterruptedScanSessions(ctx)
	if err != nil {
		slog.Warn("could not check for interrupted scans", slog.Any("error", err))
		return
	}
	for _, session := range sessions {
		slog.Warn("interrupted scan detected",
			slog.String("scan_id", session.ID),
			slog.String("status", session.Status),
			slog.Time("started_at", session.StartedAt),
		)
	}
	if len(sessions) > 0 {
		slog.Info("scheduling reconciliation is recommended for interrupted scans",
			slog.Int("count", len(sessions)))
	}
}

// handleWatchEvent applies one debounced, stability-checked filesystem event.
//
// The important guarantee here is what it does NOT do: a remove event never
// deletes a catalogue row. The record is left for reconciliation, because a
// remove can be the first half of a rename, a disconnecting share, or a folder
// being replaced. Deleting on that signal is how an index loses data.
//
// The event is stored through the SAME resolution session the scan uses, so a
// file that arrives while nobody is scanning cannot invent a work named after
// its own filename.
func handleWatchEvent(ctx context.Context, repository *db.Repository, session *db.ResolutionSession,
	cfg config.Config, event scanner.Event) {

	switch event.Kind {
	case scanner.EventRootOnline:
		slog.Info("media root online", slog.String("root", event.Path))
		return
	case scanner.EventRemoved, scanner.EventRenamed, scanner.EventDirRemoved:
		slog.Debug("filesystem change deferred to reconciliation",
			slog.String("kind", string(event.Kind)),
			slog.String("path", event.Path))
		return
	case scanner.EventDirCreated:
		slog.Debug("directory created", slog.String("path", event.Path))
		return
	}

	if event.File == nil {
		return
	}

	stats, err := session.Ingest(ctx, repository, []scanner.FileInfo{*event.File})
	if err != nil {
		slog.Warn("media ingest failed",
			slog.String("kind", string(event.Kind)),
			slog.String("path", event.Path),
			slog.Any("error", err))
		return
	}
	slog.Info("media event resolved",
		slog.String("kind", string(event.Kind)),
		slog.String("path", event.Path),
		slog.String("title", event.File.Parsed.Title),
		slog.Int("attached", stats.FilesAttached),
		slog.Int("queued_for_review", stats.QueuedForReview),
	)
}

// buildReconcileEmitter persists the output of a scheduled reconciliation sweep.
//
// It uses the same resolution session as the scan and the watcher, so a file
// rediscovered by a background sweep follows exactly the same rules as one found
// by an interactive scan.
func buildReconcileEmitter(ctx context.Context, repository *db.Repository,
	session *db.ResolutionSession) func(scanner.FileInfo) error {

	const batchSize = 256
	batch := make([]scanner.FileInfo, 0, batchSize)

	return func(file scanner.FileInfo) error {
		batch = append(batch, file)
		if len(batch) < batchSize {
			return nil
		}
		err := flushReconcileBatch(ctx, repository, session, batch)
		batch = batch[:0]
		return err
	}
}

// flushReconcileBatch resolves and stores one reconciliation batch.
func flushReconcileBatch(ctx context.Context, repository *db.Repository,
	session *db.ResolutionSession, batch []scanner.FileInfo) error {

	_, err := session.Ingest(ctx, repository, batch)
	return err
}
