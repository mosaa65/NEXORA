//go:build !windows

package scanner

import (
	"os"
	"strconv"
	"syscall"
)

// platformFileID returns the inode plus device, which is the POSIX equivalent of
// the Windows file index: stable across renames within a device, new when the
// file is replaced.
func platformFileID(info os.FileInfo) (string, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", false
	}
	return "posix:" + strconv.FormatUint(uint64(stat.Dev), 10) + ":" + strconv.FormatUint(uint64(stat.Ino), 10), true
}
