package transfer

import (
	"context"
	"os"
	"strings"
)

// Verifier checks that a transferred file matches its source according to the
// configured verification mode.
type Verifier struct {
	mode VerifyMode
}

func NewVerifier(mode VerifyMode) *Verifier {
	return &Verifier{mode: mode}
}

// Verify returns nil when the remote file at dest is considered equal to the
// source described by srcSize.
func (v *Verifier) Verify(ctx context.Context, backend TransferBackend, dest string, srcSize int64) error {
	if v.mode == VerifyStrict {
		return verifyStrict(ctx, backend, dest, srcSize)
	}
	return verifySize(ctx, backend, dest, srcSize)
}

// verifySize compares the remote size against the expected source size.
func verifySize(ctx context.Context, backend TransferBackend, dest string, srcSize int64) error {
	entry, err := backend.Stat(ctx, dest)
	if err != nil {
		return WrapTransferError(ErrVerifyFailed("تعذر التحقق من الملف بعد النسخ: "+err.Error()), err)
	}
	if entry.Size != srcSize {
		return ErrVerifyFailed(
			"حجم الملف على الجهاز (" + formatBytes(entry.Size) +
				") لا يطابق المصدر (" + formatBytes(srcSize) + ")",
		)
	}
	return nil
}

// verifyStrict currently falls back to size verification. A full content hash
// comparison can be added behind this flag later (see plan §41, §110).
func verifyStrict(ctx context.Context, backend TransferBackend, dest string, srcSize int64) error {
	return verifySize(ctx, backend, dest, srcSize)
}

// remoteResumeOffset returns how many bytes already exist on the remote side
// so a copy can resume instead of restarting. It only returns a value when the
// remote file exists and is smaller than the full source size.
func remoteResumeOffset(ctx context.Context, backend TransferBackend, dest string, srcSize int64) int64 {
	entry, err := backend.Stat(ctx, dest)
	if err != nil {
		return 0
	}
	if entry.IsDir {
		return 0
	}
	if entry.Size > 0 && entry.Size < srcSize {
		return entry.Size
	}
	return 0
}

// joinRemotePath appends a file name to a remote directory path, keeping the
// separator style of the base path.
func joinRemotePath(remoteDir, name string) string {
	if remoteDir == "" {
		return name
	}
	if strings.HasSuffix(remoteDir, "/") || strings.HasSuffix(remoteDir, "\\") {
		return remoteDir + name
	}
	if strings.Contains(remoteDir, "\\") {
		return strings.TrimRight(remoteDir, "\\") + "\\" + name
	}
	return strings.TrimRight(remoteDir, "/") + "/" + name
}

// statSource wraps os.Stat and returns the size or a classified source error.
func statSource(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
