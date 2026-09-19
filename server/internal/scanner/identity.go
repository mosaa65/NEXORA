package scanner

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// identityIndex answers the two questions incremental scanning needs:
//
//  1. "Have I seen this exact file version before?"  (fast skip)
//  2. "Was this path previously known under another name?" (rename vs create)
//
// It is populated once per scan from the persisted catalogue and is bounded by
// the number of already-indexed files, which is the data set we must compare
// against anyway. Lookups are O(1) map reads; nothing is stored per traversed
// directory, so a scan of a 10-million-file filesystem does not grow this.
// identityIndex is read concurrently by every metadata worker, so every mutable
// field is guarded. The immutable lookup maps are built once and only read
// afterwards; only `seen` is written during the scan.
type identityIndex struct {
	mu         sync.Mutex
	byPath     map[string]KnownFile
	byFileID   map[string][]KnownFile
	bySizeTime map[string][]KnownFile
	seen       map[string]struct{}
}

// newIdentityIndex builds the lookup used by change detection.
func newIdentityIndex(known []KnownFile) *identityIndex {
	index := &identityIndex{
		byPath:     make(map[string]KnownFile, len(known)),
		byFileID:   make(map[string][]KnownFile),
		bySizeTime: make(map[string][]KnownFile),
		seen:       make(map[string]struct{}, len(known)),
	}
	for _, record := range known {
		normalized := NormalizePathKey(record.Path)
		index.byPath[normalized] = record
		if record.FileID != "" {
			index.byFileID[record.FileID] = append(index.byFileID[record.FileID], record)
		}
		key := sizeTimeKey(record.Size, record.ModTime.UTC().UnixNano())
		index.bySizeTime[key] = append(index.bySizeTime[key], record)
	}
	return index
}

// changeResult describes what should happen to one scanned file.
type changeResult struct {
	Kind        ChangeKind
	Known       *KnownFile
	RenamedFrom string
}

// classify decides whether a scanned file is new, changed, unchanged or a
// rename of a record whose old path was not visited in this pass.
//
// The order matters: an exact path match is authoritative and cheapest, so it
// is checked first and covers the overwhelming majority of a full scan. Only
// when the path is unknown do we fall back to filesystem identity and then to
// a size+modification-time fingerprint to infer a move.
func (i *identityIndex) classify(path string, fingerprint Fingerprint, rootReachable bool) changeResult {
	normalized := NormalizePathKey(path)

	if known, exists := i.byPath[normalized]; exists {
		i.markSeen(normalized)
		if !rootReachable {
			// The root is unreachable mid-scan; we cannot trust what we just
			// read, so treat the record as unchanged rather than modified.
			return changeResult{Kind: ChangeUnchanged, Known: &known}
		}
		if known.Fingerprint().Equal(fingerprint) {
			return changeResult{Kind: ChangeUnchanged, Known: &known}
		}
		return changeResult{Kind: ChangeChanged, Known: &known}
	}

	// Unknown path: is this a file I already know, moved?
	// findMoved runs under the same lock that marks the old path as seen, so two
	// workers cannot both claim one old path as their rename target and create
	// two records for a single file.
	if candidate, oldPath := i.claimMoved(fingerprint); candidate != nil {
		return changeResult{Kind: ChangeRenamed, Known: candidate, RenamedFrom: oldPath}
	}

	return changeResult{Kind: ChangeNew}
}

// claimMoved atomically finds an unseen record matching the fingerprint and
// marks its old path as claimed.
func (i *identityIndex) claimMoved(fingerprint Fingerprint) (*KnownFile, string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.findMovedLocked(fingerprint)
}

// markSeenIfAbsent records a path and reports whether it was newly recorded.
// The boolean makes duplicate detection explicit at the call site.
func (i *identityIndex) markSeenIfAbsent(normalizedPath string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, exists := i.seen[normalizedPath]; exists {
		return false
	}
	i.seen[normalizedPath] = struct{}{}
	return true
}

// markSeen records that a path was observed in this pass.
func (i *identityIndex) markSeen(normalizedPath string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.seen[normalizedPath] = struct{}{}
}

// wasSeenLocked reports whether a path was already observed. The caller must
// hold the lock.
func (i *identityIndex) wasSeenLocked(normalizedPath string) bool {
	_, seen := i.seen[normalizedPath]
	return seen
}

// wasSeen reports whether a path was already observed in this pass.
func (i *identityIndex) wasSeen(normalizedPath string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	_, seen := i.seen[normalizedPath]
	return seen
}

// findMovedLocked looks for an already-indexed file with the same identity that
// was not observed at its old path during this pass. A rename is only claimed
// when the match is unambiguous: a single candidate of the same size with a
// stable filesystem ID, or, when IDs are unavailable (SMB shares, removable
// media), a single same-size, same-mtime candidate that was not seen anywhere.
//
// The caller must hold i.mu.
func (i *identityIndex) findMovedLocked(fingerprint Fingerprint) (*KnownFile, string) {
	if fingerprint.FileID != "" {
		candidates := i.byFileID[fingerprint.FileID]
		var match *KnownFile
		for idx := range candidates {
			if i.wasSeenLocked(NormalizePathKey(candidates[idx].Path)) {
				continue
			}
			if candidates[idx].Size != fingerprint.Size {
				continue
			}
			if match != nil {
				// Two unseen records share this file ID, which should be
				// impossible; refuse to guess.
				return nil, ""
			}
			match = &candidates[idx]
		}
		if match != nil {
			// Claim it inside the same critical section that found it, so a
			// concurrent worker cannot claim the same record as its own rename.
			i.seen[NormalizePathKey(match.Path)] = struct{}{}
			return match, match.Path
		}
	}

	unseen := make([]KnownFile, 0, 2)
	for _, candidate := range i.bySizeTime[sizeTimeKey(fingerprint.Size, fingerprint.ModTime)] {
		if i.wasSeenLocked(NormalizePathKey(candidate.Path)) {
			continue
		}
		unseen = append(unseen, candidate)
	}
	if len(unseen) == 1 {
		// Claim it immediately so no other worker can reuse this record.
		i.seen[NormalizePathKey(unseen[0].Path)] = struct{}{}
		return &unseen[0], unseen[0].Path
	}
	return nil, ""
}

// missing returns every indexed path under the scanned roots that was not
// observed during the pass. Filtering by root keeps a scan of Disk A from
// declaring Disk B's files missing.
func (i *identityIndex) missing(roots []string) []KnownFile {
	normalizedRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		normalizedRoots = append(normalizedRoots, NormalizePathKey(root))
	}
	missing := make([]KnownFile, 0)
	for normalized, record := range i.byPath {
		if i.wasSeen(normalized) {
			continue
		}
		if !pathUnderAnyRoot(normalized, normalizedRoots) {
			continue
		}
		missing = append(missing, record)
	}
	sort.Slice(missing, func(a, b int) bool { return missing[a].Path < missing[b].Path })
	return missing
}

func pathUnderAnyRoot(normalizedPath string, normalizedRoots []string) bool {
	if len(normalizedRoots) == 0 {
		return true
	}
	for _, root := range normalizedRoots {
		if root == "" {
			continue
		}
		if normalizedPath == root || strings.HasPrefix(normalizedPath, root+"/") {
			return true
		}
	}
	return false
}

// sizeTimeKey is the cheap fingerprint pair used for rename inference when the
// filesystem exposes no stable file ID.
func sizeTimeKey(size, modTimeNano int64) string {
	return itoa64(size) + ":" + itoa64(modTimeNano)
}

func itoa64(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}

// NormalizePathKey produces the canonical comparison key for a path. Windows is
// case-insensitive and mixes separators, and network shares can report either,
// so both are folded. This is what prevents the same file from being indexed
// twice because one pass saw `D:\Media\X.mkv` and the next `d:/media/x.mkv`.
func NormalizePathKey(path string) string {
	if path == "" {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(path))
	cleaned = strings.TrimPrefix(cleaned, "./")
	return strings.ToLower(cleaned)
}

// RootID derives a stable identifier for a media root so per-root state can be
// persisted and looked up again after a restart, even if the drive letter of an
// external disk changes.
func RootID(root string) string {
	cleaned := NormalizePathKey(root)
	volume := volumeKey(cleaned)
	if volume != "" {
		return volume
	}
	sum := sha256.Sum256([]byte(cleaned))
	return "root-" + hex.EncodeToString(sum[:8])
}

// volumeKey extracts the volume identity of a normalized path ("d:", "//nas/share").
func volumeKey(normalized string) string {
	if strings.HasPrefix(normalized, "//") {
		// UNC path: the share is the stable identity, not the deep folder.
		parts := strings.Split(strings.TrimPrefix(normalized, "//"), "/")
		if len(parts) >= 2 {
			return "//" + parts[0] + "/" + parts[1]
		}
		if len(parts) == 1 {
			return "//" + parts[0]
		}
		return ""
	}
	if len(normalized) >= 2 && normalized[1] == ':' {
		return normalized[:2]
	}
	return ""
}

// fileIDFromStat extracts a stable filesystem file identity when the platform
// provides one. It is best-effort by design: the fingerprint falls back to
// size+modTime when this returns "".
func fileIDFromStat(info os.FileInfo) string {
	if info == nil || info.Sys() == nil {
		return ""
	}
	if id, ok := platformFileID(info); ok {
		return id
	}
	// Last resort: the non-content identity the standard library always has.
	return base64.StdEncoding.EncodeToString([]byte(sizeTimeKey(info.Size(), info.ModTime().UTC().UnixNano())))
}
