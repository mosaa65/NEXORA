package transfer

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// StorageBackend transfers files to a local removable drive (e.g. USB stick)
// using plain Go file I/O. It does not need PowerShell, MTP or AFC.
type StorageBackend struct{}

func NewStorageBackend() *StorageBackend {
	return &StorageBackend{}
}

func (b *StorageBackend) Connect(ctx context.Context, device Device, destination TransferDestination) error {
	return nil
}

func (b *StorageBackend) List(ctx context.Context, path string) ([]RemoteEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]RemoteEntry, 0, len(entries))
	for _, e := range entries {
		info, ierr := e.Info()
		re := RemoteEntry{
			Name:  e.Name(),
			Path:  filepath.Join(path, e.Name()),
			IsDir: e.IsDir(),
		}
		if ierr == nil {
			re.Size = info.Size()
			re.Modified = info.ModTime()
		}
		out = append(out, re)
	}
	return out, nil
}

func (b *StorageBackend) Stat(ctx context.Context, path string) (RemoteEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return RemoteEntry{}, err
	}
	return RemoteEntry{
		Name:     filepath.Base(path),
		Path:     path,
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		Modified: info.ModTime(),
	}, nil
}

func (b *StorageBackend) Mkdir(ctx context.Context, path string) error {
	return os.MkdirAll(path, 0o755)
}

// Put streams a local source file into the remote destination file, creating
// the destination directory and its parents first.
func (b *StorageBackend) Put(ctx context.Context, source, destination string, opts PutOptions) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	targetPath := destination

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	var out *os.File
	if opts.ResumeOffset > 0 {
		out, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE, 0666)
	} else {
		out, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	}
	if err != nil {
		return err
	}
	defer out.Close()

	// Seek to the resume offset if one is provided and within file bounds.
	if opts.ResumeOffset > 0 {
		if _, err := out.Seek(opts.ResumeOffset, io.SeekStart); err != nil {
			return err
		}
		if _, err := in.Seek(opts.ResumeOffset, io.SeekStart); err != nil {
			return err
		}
	}

	bufSize := int(opts.BufferSize)
	if bufSize <= 0 {
		bufSize = 4 * 1024 * 1024 // 4 MB optimal buffer for USB sequential throughput
	}
	buf := make([]byte, bufSize)

	var transferred int64
	if opts.ResumeOffset > 0 {
		transferred = opts.ResumeOffset
	}

	for {
		select {
		case <-ctx.Done():
			_ = out.Close()
			return ctx.Err()
		default:
		}

		n, readErr := in.Read(buf)
		if n > 0 {
			w, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			transferred += int64(w)
			if opts.OnProgress != nil {
				opts.OnProgress(transferred)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return readErr
		}
	}

	// Flush dirty OS write buffers to physical flash storage before declaring done
	_ = out.Sync()
	return nil
}

func (b *StorageBackend) Delete(ctx context.Context, path string) error {
	return os.Remove(path)
}

func (b *StorageBackend) Rename(ctx context.Context, oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (b *StorageBackend) Close() error { return nil }

var _ TransferBackend = (*StorageBackend)(nil)
