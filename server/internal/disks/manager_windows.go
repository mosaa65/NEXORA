//go:build windows

package disks

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"nexora/server/internal/db"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	getLogicalDrives    = kernel32.NewProc("GetLogicalDrives")
	getDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
	getVolumeInfoW      = kernel32.NewProc("GetVolumeInformationW")
)

func detectDisks(ctx context.Context) ([]db.StorageDisk, error) {
	mask, _, _ := getLogicalDrives.Call()
	if mask == 0 {
		return nil, fmt.Errorf("could not retrieve logical drives")
	}

	now := time.Now().UTC()
	disks := make([]db.StorageDisk, 0)

	for i := 0; i < 26; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if (mask & (1 << uint(i))) == 0 {
			continue
		}

		letter := string(rune('A' + i))
		rootPath := fmt.Sprintf("%s:\\", letter)
		rootPtr, _ := syscall.UTF16PtrFromString(rootPath)

		var freeBytes, totalBytes, totalFreeBytes int64
		ret, _, _ := getDiskFreeSpaceExW.Call(
			uintptr(unsafe.Pointer(rootPtr)),
			uintptr(unsafe.Pointer(&freeBytes)),
			uintptr(unsafe.Pointer(&totalBytes)),
			uintptr(unsafe.Pointer(&totalFreeBytes)),
		)

		if ret == 0 || totalBytes == 0 {
			continue
		}

		volName := make([]uint16, 256)
		getVolumeInfoW.Call(
			uintptr(unsafe.Pointer(rootPtr)),
			uintptr(unsafe.Pointer(&volName[0])),
			uintptr(len(volName)),
			0, 0, 0, 0, 0,
		)
		label := syscall.UTF16ToString(volName)
		if label == "" {
			label = fmt.Sprintf("Local Disk (%s:)", letter)
		}

		used := totalBytes - freeBytes
		var usedPercent float64
		if totalBytes > 0 {
			usedPercent = float64(used) / float64(totalBytes) * 100.0
		}

		disks = append(disks, db.StorageDisk{
			DiskLetter:  letter,
			DiskLabel:   label,
			TotalSpace:  totalBytes,
			FreeSpace:   freeBytes,
			UsedSpace:   used,
			UsedPercent: usedPercent,
			IsActive:    true,
			LastScanned: &now,
		})
	}

	return disks, nil
}
