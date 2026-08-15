//go:build !windows

package disks

import (
	"context"
	"syscall"
	"time"

	"nexora/server/internal/db"
)

func detectDisks(ctx context.Context) ([]db.StorageDisk, error) {
	now := time.Now().UTC()
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return nil, err
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - free
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	return []db.StorageDisk{
		{
			DiskLetter:  "/",
			DiskLabel:   "Root Volume",
			TotalSpace:  total,
			FreeSpace:   free,
			UsedSpace:   used,
			UsedPercent: usedPercent,
			IsActive:    true,
			LastScanned: &now,
		},
	}, nil
}
