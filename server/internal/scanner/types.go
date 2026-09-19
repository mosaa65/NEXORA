// Package scanner turns a media filesystem into a structured, incremental,
// fault-tolerant inventory. The design separates four concerns so that none of
// them can take down the others:
//
//	Discovery  → walk directories, apply ignore rules, emit candidates
//	Metadata   → stat + parse each candidate (bounded worker pool)
//	Persistence→ caller-owned; the scanner never touches a database
//	Watch      → fast path only; reconciliation stays the source of truth
//
// A scanner is I/O bound, so it streams results through a callback instead of
// accumulating a slice. Scanner.Scan exists for small/embedded use only.
package scanner

import (
	"time"
)

// -----------------------------------------------------------------------------
// File identity and change detection
// -----------------------------------------------------------------------------

// FileState is the reconciliation lifecycle of an indexed file. It is never
// inferred from a single failed read: a transient error moves a record to
// UNAVAILABLE rather than deleting it.
type FileState string

const (
	// StatePendingScan is returned by a fingerprint comparison when the file
	// has never been recorded before.
	StatePendingScan FileState = "pending_scan"
	// StateActive means the file exists on disk and its fingerprint matches
	// the persisted one; it is skipped in incremental scans.
	StateActive FileState = "active"
	// StateChanged means the file exists but its fingerprint differs.
	StateChanged FileState = "changed"
	// StateMissing means the file was absent during a scan whose root was
	// reachable. Cleanup is a separate, explicit operation.
	StateMissing FileState = "missing"
	// StateUnavailable means the root or file could not be read (offline disk,
	// permission, network timeout). Never treated as a deletion.
	StateUnavailable FileState = "unavailable"
	// StateError means a deterministic, non-transient failure occurred.
	StateError FileState = "error"
	// StateRenamed means the same physical file was observed under a new path.
	StateRenamed FileState = "renamed"
)

// Fingerprint is the cheap identity of a file on disk. It deliberately ignores
// file content: hashing terabytes during a full scan is not acceptable. Size,
// modification time and (where the OS provides it) the filesystem file ID are
// enough to detect new/changed/unchanged files with very high accuracy.
type Fingerprint struct {
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTimeUnixNano"`
	FileID  string `json:"fileId,omitempty"`
}

// Equal reports whether two fingerprints describe the same file version.
// When a stable filesystem file ID exists it is the strongest signal, so a
// surviving FileID with unchanged size wins over a touched ModTime (some
// filesystems and network shares rewrite mtimes without changing content).
func (a Fingerprint) Equal(b Fingerprint) bool {
	if a.FileID != "" && b.FileID != "" {
		return a.FileID == b.FileID && a.Size == b.Size
	}
	return a.Size == b.Size && a.ModTime == b.ModTime
}

// ChangeKind classifies the delta between a scanned file and its persisted
// record. It drives both the incremental skip and the scan report counters.
type ChangeKind string

const (
	ChangeNew       ChangeKind = "new"
	ChangeChanged   ChangeKind = "changed"
	ChangeUnchanged ChangeKind = "unchanged"
	ChangeRenamed   ChangeKind = "renamed"
	ChangeRemoved   ChangeKind = "removed"
)

// KnownFile is the persisted record the reconciler compares against. It is a
// plain value so the scanner package stays free of any database dependency.
type KnownFile struct {
	ID          int64
	Path        string
	Size        int64
	ModTime     time.Time
	FileID      string
	RootID      string
	Parsed      ParsedName
	LastScanID  string
	LastSeenAt  time.Time
	State       FileState
	ContentHash string
}

// Fingerprint returns the comparable fingerprint of a known record.
func (k KnownFile) Fingerprint() Fingerprint {
	return Fingerprint{Size: k.Size, ModTime: k.ModTime.UTC().UnixNano(), FileID: k.FileID}
}
