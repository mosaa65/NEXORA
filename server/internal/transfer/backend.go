package transfer

import (
	"context"
	"time"
)

// defaultBufferSize is the streaming buffer used by backends that write via
// plain Go I/O. Starting value is 4 MB (see development plan, §33); it should
// be tuned after a real benchmark on the target hardware.
const defaultBufferSize = 4 * 1024 * 1024

// TransferDestination is a unified, backend-agnostic description of where files
// should be copied on a device. It replaces the scattered SubFolder/TargetFolder
// fields so the engine never depends on per-device conventions.
type TransferDestination struct {
	DeviceID   string
	DeviceType DeviceType

	// AppID is used only for iOS (the application Bundle ID to receive files).
	AppID string

	// RemotePath is the folder on the device where files are copied, e.g.
	//   iOS:     /Documents/Series/Breaking Bad/Season 1
	//   Android: Internal Storage/Movies/Breaking Bad/Season 1
	//   Storage: E:\Movies\Breaking Bad\Season 1
	RemotePath string
}

// RemoteEntry is a single item (folder or file) inside a device filesystem,
// returned by backend listing operations.
type RemoteEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified,omitempty"`
}

// PutOptions controls how a backend writes a single source file to a device.
type PutOptions struct {
	BufferSize   int64
	ResumeOffset int64
	Overwrite    bool

	// OnProgress is invoked with the cumulative number of bytes written so far
	// for this file. It is called from the backend's writing goroutine and must
	// be safe to call concurrently.
	OnProgress func(transferred int64)
}

// TransferBackend is the unified interface every device transport implements:
// iOS (AFC/HouseArrest), Android (MTP), and local removable Storage.
type TransferBackend interface {
	// Connect establishes any persistent resources needed to talk to the
	// device (e.g. a long-lived go-ios AFC connection for iOS).
	Connect(ctx context.Context, device Device, destination TransferDestination) error

	// List returns the entries located directly under path.
	List(ctx context.Context, path string) ([]RemoteEntry, error)

	// Stat returns metadata for a single path.
	Stat(ctx context.Context, path string) (RemoteEntry, error)

	// Mkdir creates a directory (and any missing parents) at path.
	Mkdir(ctx context.Context, path string) error

	// Put streams a local source file to the remote destination path.
	Put(ctx context.Context, source, destination string, opts PutOptions) error

	// Delete removes a remote path.
	Delete(ctx context.Context, path string) error

	// Rename moves/renames a remote path.
	Rename(ctx context.Context, oldPath, newPath string) error

	// Close releases all persistent resources held by the backend.
	Close() error
}
