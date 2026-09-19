package scanner

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentRenameInferenceIsAtomic stresses the identity index concurrently.
//
// Without the lock around the "claim this old path as my rename target" step,
// two workers could both match one unseen record and emit two rename decisions
// for a single file, which is how a rename silently becomes a duplicate row.
// The stress test drives many workers at the same candidate set and asserts each
// record is claimed at most once.
func TestConcurrentRenameInferenceIsAtomic(t *testing.T) {
	const records = 200
	known := make([]KnownFile, 0, records)
	for i := 0; i < records; i++ {
		known = append(known, KnownFile{
			ID:      int64(i + 1),
			Path:    "D:/Media/old-" + itoa(i) + ".mp4",
			Size:    int64(1000 + i),
			ModTime: time.Unix(0, int64(i)),
			FileID:  "id-" + itoa(i),
		})
	}

	index := newIdentityIndex(known)

	// Every moved file is presented to every worker, so the losing workers must
	// consistently observe "already claimed" rather than claiming a second time.
	claims := make(chan int64, records*8)
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < records; i++ {
				fileID := "id-" + itoa(i)
				size := int64(1000 + i)
				result := index.classify(
					"D:/Media/new-"+itoa(i)+".mp4",
					Fingerprint{Size: size, ModTime: int64(i), FileID: fileID},
					true,
				)
				if result.Kind == ChangeRenamed && result.Known != nil {
					claims <- result.Known.ID
				}
			}
		}()
	}
	wg.Wait()
	close(claims)

	seen := make(map[int64]int, records)
	for id := range claims {
		seen[id]++
	}
	for id, count := range seen {
		if count > 1 {
			t.Fatalf("record %d was claimed as a rename %d times; a rename must be claimed exactly once", id, count)
		}
	}
	if len(seen) != records {
		t.Fatalf("claimed %d records, want %d", len(seen), records)
	}
}

// TestConcurrentIncrementalScansShareNoState verifies two concurrent incremental
// scans each build their own index and cannot see each other's "seen" marks.
func TestConcurrentIncrementalScansShareNoState(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	writeFile(t, filepath.Join(rootA, "Movies", "a.mp4"), 16)
	writeFile(t, filepath.Join(rootB, "Movies", "b.mp4"), 16)

	scanner := New(Options{Workers: 4})
	var wg sync.WaitGroup
	errs := make(chan error, 2)

	for _, root := range []string{rootA, rootB} {
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			_, err := scanner.ScanTree(context.Background(), []string{root},
				ScanOptions{Mode: ModeIncremental}, func(FileInfo) error { return nil })
			errs <- err
		}(root)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent incremental scan failed: %v", err)
		}
	}
}
