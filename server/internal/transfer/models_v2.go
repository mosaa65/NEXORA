package transfer

import "time"

// TransferPhase describes the fine-grained stage a job is in.
type TransferPhase string

const (
	PhaseQueued               TransferPhase = "queued"
	PhasePreparing            TransferPhase = "preparing"
	PhaseConnecting           TransferPhase = "connecting"
	PhaseCheckingDestination  TransferPhase = "checking_destination"
	PhaseCopying              TransferPhase = "copying"
	PhaseRetrying             TransferPhase = "retrying"
	PhaseVerifying            TransferPhase = "verifying"
	PhaseWaitingDevice        TransferPhase = "waiting_device"
	PhaseCompleted            TransferPhase = "completed"
	PhaseFailed               TransferPhase = "failed"
	PhaseCancelled            TransferPhase = "cancelled"
	PhasePaused               TransferPhase = "paused"
)

// TransferFile is a single file within a multi-file job.
type TransferFile struct {
	ID string

	SourcePath      string
	DestinationPath string

	Size        int64
	Transferred int64

	Status JobStatus

	RetryCount int

	ResumeOffset int64

	StartedAt   time.Time
	CompletedAt *time.Time
}

// TransferJobV2 is the expanded, multi-file job model used by the v2 engine.
// It lives alongside the legacy single-file TransferJob so the existing
// REST/UI contract is preserved until the front end moves to multi-file jobs.
type TransferJobV2 struct {
	ID string

	DeviceID   string
	DeviceName string
	DeviceType DeviceType

	Destination TransferDestination

	Files []TransferFile

	TotalBytes       int64
	TransferredBytes int64

	CurrentFileIndex int
	CurrentFile      string

	Progress float64
	SpeedBps int64
	ETA      time.Duration

	Status JobStatus
	Phase  TransferPhase

	RetryCount int

	Error *TransferError

	StartedAt   time.Time
	CompletedAt *time.Time
}
