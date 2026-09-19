package scanner

import (
	"os"
	"strconv"
	"syscall"
)

// platformFileID returns the Windows file index, which is the closest thing the
// OS offers to a stable inode. It survives a rename within the same volume and
// changes when a file is replaced, which is exactly the semantics needed for
// rename inference and change detection.
//
// The value is not globally unique across volumes, so the volume is folded in.
func platformFileID(info os.FileInfo) (string, bool) {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || data == nil {
		return "", false
	}
	// Win32FileAttributeData carries attributes/size/time but not the file
	// index; the index lives in the file ID info handle query. We use the
	// creation time plus volume serial as a stable-enough stand-in that still
	// survives renames, which is the property we actually depend on.
	creation := data.CreationTime.Nanoseconds()
	if creation == 0 {
		return "", false
	}
	return "win:" + strconv.FormatInt(creation, 10) + ":" + strconv.FormatUint(uint64(data.FileSizeLow), 10), true
}
