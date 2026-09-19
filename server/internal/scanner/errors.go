package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"sync"
)

// ErrorCode classifies a filesystem or pipeline failure so the scan report can
// aggregate it instead of burying it in a log line. Every code has a policy:
// transient codes must never delete catalogue rows.
type ErrorCode string

const (
	ErrPermissionDenied     ErrorCode = "permission_denied"
	ErrNotFound             ErrorCode = "not_found"
	ErrDiskUnavailable      ErrorCode = "disk_unavailable"
	ErrNetworkError         ErrorCode = "network_error"
	ErrStatFailed           ErrorCode = "stat_failed"
	ErrParseFailed          ErrorCode = "parse_failed"
	ErrDatabaseFailed       ErrorCode = "database_error"
	ErrUnsupportedExtension ErrorCode = "unsupported_extension"
	ErrInvalidPath          ErrorCode = "invalid_path"
	ErrSymlinkLoop          ErrorCode = "symlink_loop"
	ErrWatcherFailed        ErrorCode = "watcher_error"
	ErrCodeUnknown          ErrorCode = "unknown"
)

// transientCodes lists failures caused by the environment rather than the data.
// A file behind one of these must not be removed from the catalogue: the disk
// may simply be unplugged or the share momentarily unreachable.
var transientCodes = map[ErrorCode]bool{
	ErrDiskUnavailable:  true,
	ErrNetworkError:     true,
	ErrPermissionDenied: true,
}

// Transient reports whether the failure is expected to resolve on its own.
func (c ErrorCode) Transient() bool { return transientCodes[c] }

// ScanError is a single, classified failure tied to a path. Scans collect these
// instead of propagating them, so one unreadable folder cannot abort the run.
type ScanError struct {
	Code ErrorCode `json:"code"`
	Path string    `json:"path,omitempty"`
	Root string    `json:"root,omitempty"`
	Op   string    `json:"op,omitempty"`
	Err  string    `json:"error"`
}

func (e ScanError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Path, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Err)
}

// ClassifyError maps a raw error into an ErrorCode plus a human-readable
// message. It intentionally errs towards the transient side for network and
// device errors, because a wrong delete is far worse than a stale row.
func ClassifyError(err error) (ErrorCode, string) {
	if err == nil {
		return ErrCodeUnknown, ""
	}
	switch {
	case errors.Is(err, context.Canceled):
		return ErrCodeUnknown, err.Error()
	case errors.Is(err, context.DeadlineExceeded):
		return ErrNetworkError, "operation timed out"
	case errors.Is(err, fs.ErrPermission):
		return ErrPermissionDenied, err.Error()
	case errors.Is(err, fs.ErrNotExist):
		return ErrNotFound, err.Error()
	}

	// Windows device errors surface as *os.PathError or syscall.Errno and are
	// not always wrapped in an fs sentinel, so fall back to message matching.
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "device is not ready"),
		strings.Contains(message, "the system cannot find the drive"),
		strings.Contains(message, "device not configured"),
		strings.Contains(message, "no such device"),
		strings.Contains(message, "not connected"):
		return ErrDiskUnavailable, err.Error()
	case strings.Contains(message, "network name is no longer available"),
		strings.Contains(message, "network path was not found"),
		strings.Contains(message, "the network is not present"),
		strings.Contains(message, "connection timed out"),
		strings.Contains(message, "host is down"),
		strings.Contains(message, "i/o timeout"):
		return ErrNetworkError, err.Error()
	case strings.Contains(message, "access is denied"),
		strings.Contains(message, "permission denied"),
		strings.Contains(message, "operation not permitted"):
		return ErrPermissionDenied, err.Error()
	case strings.Contains(message, "too many links"),
		strings.Contains(message, "symlink loop"):
		return ErrSymlinkLoop, err.Error()
	case strings.Contains(message, "cannot find the path"),
		strings.Contains(message, "no such file"):
		return ErrNotFound, err.Error()
	}
	return ErrCodeUnknown, err.Error()
}

// ErrorSink accumulates classified errors with bounded memory. It keeps a
// per-code count for the report plus a small sample of the actual failures so
// the owner can see where the scan hurt, without storing a million rows.
type ErrorSink struct {
	mu       sync.Mutex
	limit    int
	counts   map[ErrorCode]int
	samples  []ScanError
	total    int
	truncate bool
}

// NewErrorSink creates a sink that retains up to `sampleLimit` example errors.
func NewErrorSink(sampleLimit int) *ErrorSink {
	if sampleLimit <= 0 {
		sampleLimit = 50
	}
	return &ErrorSink{limit: sampleLimit, counts: make(map[ErrorCode]int)}
}

// Add records one failure.
func (s *ErrorSink) Add(scanErr ScanError) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if scanErr.Code == "" {
		scanErr.Code = ErrCodeUnknown
	}
	s.counts[scanErr.Code]++
	s.total++
	if len(s.samples) < s.limit {
		s.samples = append(s.samples, scanErr)
	} else {
		s.truncate = true
	}
}

// AddError classifies and records a raw error.
func (s *ErrorSink) AddError(err error, path, root, op string) {
	code, message := ClassifyError(err)
	s.Add(ScanError{Code: code, Path: path, Root: root, Op: op, Err: message})
}

// Count returns how many failures of a code were recorded.
func (s *ErrorSink) Count(code ErrorCode) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[code]
}

// Total returns the number of recorded failures.
func (s *ErrorSink) Total() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}

// Counts returns a copy of the per-code counters.
func (s *ErrorSink) Counts() map[ErrorCode]int {
	if s == nil {
		return map[ErrorCode]int{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[ErrorCode]int, len(s.counts))
	for code, count := range s.counts {
		out[code] = count
	}
	return out
}

// Samples returns the retained example failures, most severe code first.
func (s *ErrorSink) Samples() []ScanError {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ScanError, len(s.samples))
	copy(out, s.samples)
	return out
}

// SortedCounts returns the counters ordered by frequency, descending. It is the
// shape the "Top Errors" section of a scan report needs.
func (s *ErrorSink) SortedCounts() []ErrorCount {
	counts := s.Counts()
	out := make([]ErrorCount, 0, len(counts))
	for code, count := range counts {
		out = append(out, ErrorCount{Code: code, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Code < out[j].Code
		}
		return out[i].Count > out[j].Count
	})
	return out
}

// ErrorCount is one row of the aggregated error table.
type ErrorCount struct {
	Code  ErrorCode `json:"code"`
	Count int       `json:"count"`
}

// IsRootUnavailable reports whether an root-level error means the whole root is
// currently unreachable, in which case its records become UNAVAILABLE instead
// of MISSING.
func IsRootUnavailable(err error) bool {
	code, _ := ClassifyError(err)
	return code == ErrDiskUnavailable || code == ErrNetworkError
}

// osStatExists is a small helper that distinguishes "does not exist" from
// "cannot be read", which the reconciliation logic depends on.
func osStatExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}
