package transfer

import (
	"context"
	"time"
)

type DeviceType string

const (
	DeviceAndroid DeviceType = "android"
	DeviceIOS     DeviceType = "ios"
	DeviceStorage DeviceType = "storage"
)

type Device struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Model      string     `json:"model"`
	Type       DeviceType `json:"type"`
	Status     string     `json:"status"`
	FreeSpace  int64      `json:"free_space"`
	TotalSpace int64      `json:"total_space"`
	FileSystem string     `json:"file_system,omitempty"`
}

type DeviceApp struct {
	BundleID           string `json:"bundle_id"`
	Name               string `json:"name"`
	Icon               string `json:"icon"`
	FileSharingEnabled bool   `json:"file_sharing_enabled"`
	Category           string `json:"category"`
}

type AppFolder struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"is_dir"`
	Children []AppFolder `json:"children,omitempty"`
}

type windowsPortableRecord struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	Type       string `json:"Type"`
	Status     string `json:"Status"`
	FreeSpace  int64  `json:"FreeSpace"`
	TotalSpace int64  `json:"TotalSpace"`
}

type CopyRequest struct {
	DeviceID     string   `json:"device_id"`
	SourcePath   string   `json:"source_path"`
	SourcePaths  []string `json:"source_paths,omitempty"` // multi-file (Phase 4)
	TargetApp    string   `json:"target_app,omitempty"`   // For iOS (e.g. org.videolan.vlc-ios)
	MediaID      int64    `json:"media_id,omitempty"`
	FileID       int64    `json:"file_id,omitempty"`
	SubFolder    string   `json:"sub_folder,omitempty"`    // e.g. "Attack on Titan"
	TargetFolder string   `json:"target_folder,omitempty"` // e.g. "Movies" or "Download"
}

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
	StatusPaused     JobStatus = "paused"
	StatusWaiting    JobStatus = "waiting_device"
	StatusRetrying   JobStatus = "retrying"
)

type TransferJob struct {
	ID               string      `json:"id"`
	DeviceID         string      `json:"device_id"`
	DeviceName       string      `json:"device_name"`
	DeviceType       string      `json:"device_type"`
	SourcePath       string      `json:"source_path"`
	FileName         string      `json:"file_name"`
	CurrentFile      string      `json:"current_file,omitempty"`
	FileSize         int64       `json:"file_size"`
	TotalBytes       int64       `json:"total_bytes,omitempty"`
	TransferredBytes int64       `json:"transferred_bytes,omitempty"`
	FileCount        int64       `json:"file_count,omitempty"`
	TotalFiles       int64       `json:"total_files,omitempty"`
	FinishedFiles    int64       `json:"finished_files,omitempty"`
	CompletedFiles   int64       `json:"completed_files,omitempty"`
	Transferred      int64       `json:"transferred"`
	Progress         float64     `json:"progress"` // 0.0 to 100.0
	SpeedMBps        float64     `json:"speed_mbps"`
	SpeedBps         int64       `json:"speed_bps,omitempty"`
	ETASeconds       int64       `json:"eta_seconds,omitempty"`
	DestinationPath  string      `json:"destination_path,omitempty"`
	Phase            string      `json:"phase,omitempty"`
	Status           JobStatus   `json:"status"`
	Error            string      `json:"error,omitempty"`
	StartedAt        time.Time   `json:"started_at"`
	CompletedAt      *time.Time  `json:"completed_at,omitempty"`
	cancelFunc       context.CancelFunc
}

type Options struct {
	AndroidTargetFolder string
	IOSBundleID         string
}
