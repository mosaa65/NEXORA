package copybridge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nexora/server/internal/transfer"
)

type CopyRequest struct {
	DeviceID     string   `json:"device_id"`
	SourceURL    string   `json:"source_url,omitempty"`
	SourceURLs   []string `json:"source_urls,omitempty"`
	SourcePath   string   `json:"source_path,omitempty"`
	SourcePaths  []string `json:"source_paths,omitempty"`
	FileName     string   `json:"file_name,omitempty"`
	FileNames    []string `json:"file_names,omitempty"`
	SourceSize   int64    `json:"source_size,omitempty"`
	SourceSizes  []int64  `json:"source_sizes,omitempty"`
	FileID       int64    `json:"file_id,omitempty"`
	FileIDs      []int64  `json:"file_ids,omitempty"`
	TargetApp    string   `json:"target_app,omitempty"`
	SubFolder    string   `json:"sub_folder,omitempty"`
	TargetFolder string   `json:"target_folder,omitempty"`
}

type sourceSpec struct {
	URL    string
	Local  string
	Name   string
	Size   int64
	Index  int
	FileID int64
}

type Service struct {
	cfg      Config
	transfer *transfer.Service
	client   *http.Client
	events   *eventBroker

	mu      sync.RWMutex
	jobs    map[string]*transfer.TransferJob
	cancels map[string]context.CancelFunc
	history []*transfer.TransferJob
}

func NewService(cfg Config) (*Service, error) {
	if strings.TrimSpace(cfg.TempDir) == "" {
		cfg.TempDir = filepath.Join(os.TempDir(), "nexora-copybridge")
	}
	if err := os.MkdirAll(cfg.TempDir, 0o755); err != nil {
		return nil, fmt.Errorf("create copy bridge temp dir: %w", err)
	}
	if err := cleanupTempRoot(cfg.TempDir); err != nil {
		return nil, fmt.Errorf("cleanup copy bridge temp dir: %w", err)
	}

	svc := &Service{
		cfg: cfg,
		transfer: transfer.NewService(transfer.Options{
			AndroidTargetFolder: cfg.AndroidTargetFolder,
			IOSBundleID:         cfg.IOSBundleID,
		}),
		client:  &http.Client{Timeout: 0},
		events:  newEventBroker(),
		jobs:    make(map[string]*transfer.TransferJob),
		cancels: make(map[string]context.CancelFunc),
		history: make([]*transfer.TransferJob, 0, 32),
	}

	go svc.relayDeviceEvents()
	return svc, nil
}

func (s *Service) Close() {
	s.transfer.Close()
}

func (s *Service) relayDeviceEvents() {
	stream, unsubscribe := s.transfer.SubscribeEvents(256)
	defer unsubscribe()
	for event := range stream {
		if event.Type == transfer.EventDevices {
			s.events.publish(event)
		}
	}
}

func (s *Service) ListDevices(ctx context.Context) ([]transfer.Device, error) {
	return s.transfer.ListDevices(ctx)
}

func (s *Service) SnapshotDevices() []transfer.Device {
	return s.transfer.SnapshotDevices()
}

func (s *Service) ListDeviceApps(ctx context.Context, deviceID string) ([]transfer.DeviceApp, error) {
	return s.transfer.ListDeviceApps(ctx, deviceID)
}

func (s *Service) ListAppFolders(ctx context.Context, deviceID, bundleID string) ([]transfer.AppFolder, error) {
	return s.transfer.ListAppFolders(ctx, deviceID, bundleID)
}

func (s *Service) ListDevicePath(ctx context.Context, deviceID, remotePath string, deviceType transfer.DeviceType, appID string) ([]transfer.RemoteEntry, error) {
	return s.transfer.ListDevicePath(ctx, deviceID, remotePath, deviceType, appID)
}

func (s *Service) CreateDeviceFolder(ctx context.Context, deviceID, remotePath string, deviceType transfer.DeviceType, appID string) error {
	return s.transfer.CreateDeviceFolder(ctx, deviceID, remotePath, deviceType, appID)
}

func (s *Service) EjectDevice(ctx context.Context, deviceID string) error {
	return s.transfer.EjectDevice(ctx, deviceID)
}

func (s *Service) StartCopy(ctx context.Context, req CopyRequest) (*transfer.TransferJob, error) {
	sources, totalBytes, err := normalizeSources(req)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.DeviceID) == "" {
		return nil, errors.New("معرف الجهاز مطلوب")
	}

	deviceName := req.DeviceID
	deviceType := inferDeviceType(req.DeviceID)
	if devices, listErr := s.transfer.ListDevices(ctx); listErr == nil {
		for _, device := range devices {
			if device.ID == req.DeviceID {
				if strings.TrimSpace(device.Name) != "" {
					deviceName = device.Name
				}
				deviceType = string(device.Type)
				break
			}
		}
	}

	jobID := fmt.Sprintf("bridge_%d_%s", time.Now().UnixNano(), safeStageName(primarySourceName(sources)))
	jobCtx, cancel := context.WithCancel(context.Background())
	job := &transfer.TransferJob{
		ID:         jobID,
		DeviceID:   req.DeviceID,
		DeviceName: deviceName,
		DeviceType: deviceType,
		SourcePath: primarySourcePath(sources),
		FileName:   primarySourceName(sources),
		FileSize:   totalBytes,
		TotalBytes: totalBytes,
		FileCount:  int64(len(sources)),
		TotalFiles: int64(len(sources)),
		Progress:   0,
		Phase:      "queued",
		Status:     transfer.StatusPending,
		StartedAt:  time.Now(),
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.cancels[jobID] = cancel
	s.history = append([]*transfer.TransferJob{job}, s.history...)
	if len(s.history) > 100 {
		s.history = s.history[:100]
	}
	s.mu.Unlock()
	s.publishJob(job)

	go s.runBridgeJob(jobCtx, jobID, req, sources, totalBytes)
	jobCopy, ok := s.GetJob(jobID)
	if !ok {
		return job, nil
	}
	return jobCopy, nil
}

func normalizeSources(req CopyRequest) ([]sourceSpec, int64, error) {
	if items := sourcesFromURLs(req); len(items) > 0 {
		return items, sumSourceSizes(items), nil
	}
	if items := sourcesFromPaths(req); len(items) > 0 {
		return items, sumSourceSizes(items), nil
	}
	return nil, 0, errors.New("source_url أو source_path مطلوبان")
}

func sourcesFromURLs(req CopyRequest) []sourceSpec {
	urls := make([]string, 0, len(req.SourceURLs)+1)
	for _, value := range req.SourceURLs {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			urls = append(urls, trimmed)
		}
	}
	if len(urls) == 0 {
		if trimmed := strings.TrimSpace(req.SourceURL); trimmed != "" {
			urls = append(urls, trimmed)
		}
	}
	if len(urls) == 0 {
		return nil
	}
	result := make([]sourceSpec, 0, len(urls))
	for index, raw := range urls {
		name := indexedValue(req.FileNames, index)
		if name == "" && index == 0 {
			name = strings.TrimSpace(req.FileName)
		}
		size := indexedSize(req.SourceSizes, index)
		if size <= 0 && index == 0 {
			size = req.SourceSize
		}
		fileID := indexedInt64(req.FileIDs, index)
		if fileID <= 0 && index == 0 {
			fileID = req.FileID
		}
		result = append(result, sourceSpec{URL: raw, Name: name, Size: size, Index: index, FileID: fileID})
	}
	return result
}

func sourcesFromPaths(req CopyRequest) []sourceSpec {
	paths := make([]string, 0, len(req.SourcePaths)+1)
	for _, value := range req.SourcePaths {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	if len(paths) == 0 {
		if trimmed := strings.TrimSpace(req.SourcePath); trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	result := make([]sourceSpec, 0, len(paths))
	for index, raw := range paths {
		name := fileNameFromAnyPath(raw)
		if explicit := indexedValue(req.FileNames, index); explicit != "" {
			name = explicit
		} else if index == 0 && strings.TrimSpace(req.FileName) != "" {
			name = strings.TrimSpace(req.FileName)
		}
		size := indexedSize(req.SourceSizes, index)
		if size <= 0 && index == 0 {
			size = req.SourceSize
		}
		if size <= 0 {
			if info, err := os.Stat(raw); err == nil {
				size = info.Size()
			}
		}
		fileID := indexedInt64(req.FileIDs, index)
		if fileID <= 0 && index == 0 {
			fileID = req.FileID
		}
		result = append(result, sourceSpec{Local: raw, Name: name, Size: size, Index: index, FileID: fileID})
	}
	return result
}

func (s *Service) runBridgeJob(ctx context.Context, jobID string, req CopyRequest, sources []sourceSpec, totalBytes int64) {
	tempDir := filepath.Join(s.cfg.TempDir, jobID)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		s.failJob(jobID, err)
		return
	}
	defer os.RemoveAll(tempDir)

	completedBytes := int64(0)
	for index, source := range sources {
		select {
		case <-ctx.Done():
			s.cancelJobTerminal(jobID)
			return
		default:
		}

		resolvedPath := source.Local
		size := source.Size
		spanStart, spanEnd := progressSpan(index, sources, completedBytes, totalBytes)

		if resolvedPath == "" && source.URL != "" {
			// Phase 2 Upgrade: Zero-Spool Direct Stream (Stream Pipe)
			// instead of staging to local disk.
			displayName := displaySourceName(source)
			s.updateJob(jobID, func(job *transfer.TransferJob) {
				job.Status = transfer.StatusProcessing
				job.Phase = "streaming"
				job.CurrentFile = displayName
				job.FileName = displayName
			})

			err := s.streamRemoteToDevice(ctx, jobID, source, req, completedBytes, spanStart, spanEnd)
			if err != nil {
				s.failJob(jobID, err)
				return
			}
			completedBytes += maxInt64(source.Size, 0)
			continue
		}

		if resolvedPath == "" {
			// URL sources are handled by streamRemoteToDevice above; reaching
			// this point means the source has neither a resolvable local path
			// nor a streamable URL. Download-then-copy staging was removed on
			// purpose, so fail loudly instead of silently falling back.
			s.failJob(jobID, fmt.Errorf("لا يمكن تحديد مصدر الملف: %s", displaySourceName(source)))
			return
		}

		if size <= 0 {
			if info, err := os.Stat(resolvedPath); err == nil {
				size = info.Size()
			}
		}
		if size > 0 && source.Size <= 0 {
			source.Size = size
		}
		if totalBytes <= 0 {
			totalBytes = recomputeTotalBytes(sources)
			if totalBytes <= 0 {
				totalBytes = size
			}
		}

		child, err := s.transfer.StartCopy(context.Background(), transfer.CopyRequest{
			DeviceID:     req.DeviceID,
			SourcePath:   resolvedPath,
			TargetApp:    req.TargetApp,
			SubFolder:    req.SubFolder,
			TargetFolder: req.TargetFolder,
		})
		if err != nil {
			s.failJob(jobID, err)
			return
		}

		if err := s.followChildJob(ctx, jobID, child.ID, displaySourceName(source), completedBytes, size, index, len(sources), totalBytes, spanStart, spanEnd); err != nil {
			s.failJob(jobID, err)
			return
		}

		completedBytes += maxInt64(size, 0)
	}

	now := time.Now()
	s.updateJob(jobID, func(job *transfer.TransferJob) {
		job.Status = transfer.StatusCompleted
		job.Phase = "completed"
		job.Progress = 100
		job.Transferred = maxInt64(totalBytes, completedBytes)
		job.TransferredBytes = maxInt64(totalBytes, completedBytes)
		job.CompletedAt = &now
		job.SpeedBps = 0
		job.SpeedMBps = 0
		job.ETASeconds = 0
		job.FinishedFiles = int64(len(sources))
		job.CompletedFiles = int64(len(sources))
	})
}

func (s *Service) streamRemoteToDevice(ctx context.Context, jobID string, source sourceSpec, req CopyRequest, completedBytes int64, spanStart, spanEnd float64) error {
	parsed, err := url.Parse(source.URL)
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("source_url غير صالح: %s", source.URL)
	}

	hReq, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(hReq)
	if err != nil {
		return fmt.Errorf("تعذر فتح اتصال مع السيرفر: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("فشل جلب الملف من السيرفر: HTTP %d", resp.StatusCode)
	}

	expectedSize := source.Size
	if expectedSize <= 0 && resp.ContentLength > 0 {
		expectedSize = resp.ContentLength
	}

	displayName := displaySourceName(source)

	deviceType := transfer.DeviceType(inferDeviceType(req.DeviceID))
	remoteDir, remotePath := remoteTargetPath(deviceType, req, displayName)

	backend, dest, err := s.transfer.GetBackendFor(ctx, req.DeviceID, req.TargetApp, deviceType)
	if err != nil {
		return err
	}
	defer backend.Close()

	if err := backend.Connect(ctx, transfer.Device{ID: req.DeviceID, Type: deviceType}, dest); err != nil {
		return err
	}

	// Create the destination folder before writing. Storage/PutStream and the
	// Android MTP script create paths lazily, but iOS AFC Open fails when the
	// parent directory does not exist, so every backend gets an idempotent
	// Mkdir ahead of the stream.
	if err := backend.Mkdir(ctx, remoteDir); err != nil {
		return err
	}

	startedAt := time.Now()
	lastTick := time.Time{}

	putOpts := transfer.PutOptions{
		BufferSize: 4 * 1024 * 1024,
		OnProgress: func(transferred int64) {
			now := time.Now()
			if lastTick.IsZero() || now.Sub(lastTick) >= 350*time.Millisecond {
				progress := spanStart
				if expectedSize > 0 {
					ratio := float64(transferred) / float64(expectedSize)
					progress = spanStart + (spanEnd-spanStart)*ratio
				}
				speedBps := int64(0)
				etaSeconds := int64(0)
				if elapsed := now.Sub(startedAt); elapsed > 0 {
					speedBps = int64(float64(transferred) / elapsed.Seconds())
					if speedBps > 0 && expectedSize > transferred {
						etaSeconds = int64(float64(expectedSize-transferred) / float64(speedBps))
					}
				}
				s.updateJob(jobID, func(job *transfer.TransferJob) {
					job.Status = transfer.StatusProcessing
					job.Phase = "copying"
					job.Progress = progress
					job.Transferred = completedBytes + transferred
					job.TransferredBytes = completedBytes + transferred
					job.SpeedBps = speedBps
					job.SpeedMBps = float64(speedBps) / (1024 * 1024)
					job.ETASeconds = etaSeconds
				})
				lastTick = now
			}
		},
	}

	return backend.PutStream(ctx, resp.Body, expectedSize, remotePath, putOpts)
}

func (s *Service) followChildJob(ctx context.Context, parentJobID, childJobID, fileName string, completedBytes, fileSize int64, index, totalFiles int, totalBytes int64, spanStart, spanEnd float64) error {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.transfer.CancelJob(childJobID)
			return ctx.Err()
		case <-ticker.C:
			child, ok := s.transfer.GetJob(childJobID)
			if !ok {
				continue
			}

			copyProgress := clampFloat(child.Progress/100, 0, 1)
			mappedProgress := spanStart + (spanEnd-spanStart)*(0.25+0.75*copyProgress)
			mappedTransferred := completedBytes + clampInt64(child.Transferred, 0, maxInt64(fileSize, child.Transferred))
			if mappedTransferred > totalBytes && totalBytes > 0 {
				mappedTransferred = totalBytes
			}
			phase := child.Phase
			if phase == "" {
				phase = "copying"
			}

			s.updateJob(parentJobID, func(job *transfer.TransferJob) {
				job.Status = child.Status
				job.Phase = phase
				job.CurrentFile = fileName
				job.FileName = fileName
				job.Progress = mappedProgress
				job.Transferred = mappedTransferred
				job.TransferredBytes = mappedTransferred
				job.SpeedBps = child.SpeedBps
				job.SpeedMBps = child.SpeedMBps
				job.ETASeconds = child.ETASeconds
				job.DestinationPath = child.DestinationPath
				job.FinishedFiles = int64(index)
				job.CompletedFiles = int64(index)
				job.FileCount = int64(totalFiles)
				job.TotalFiles = int64(totalFiles)
			})

			switch child.Status {
			case transfer.StatusCompleted:
				s.updateJob(parentJobID, func(job *transfer.TransferJob) {
					job.Status = transfer.StatusProcessing
					job.Phase = "copying"
					job.Progress = spanEnd
					job.Transferred = completedBytes + maxInt64(fileSize, child.Transferred)
					job.TransferredBytes = completedBytes + maxInt64(fileSize, child.Transferred)
					job.FinishedFiles = int64(index + 1)
					job.CompletedFiles = int64(index + 1)
				})
				return nil
			case transfer.StatusCancelled:
				return context.Canceled
			case transfer.StatusFailed:
				if strings.TrimSpace(child.Error) != "" {
					return errors.New(child.Error)
				}
				return fmt.Errorf("فشلت عملية النسخ إلى الجهاز")
			}
		}
	}
}

func (s *Service) GetJob(jobID string) (*transfer.TransferJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.jobs[jobID]
	if !exists {
		return nil, false
	}
	cp := *job
	return &cp, true
}

func (s *Service) ListJobs() []*transfer.TransferJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*transfer.TransferJob, len(s.history))
	for i, item := range s.history {
		cp := *item
		result[i] = &cp
	}
	return result
}

func (s *Service) CancelJob(jobID string) bool {
	s.mu.Lock()
	if cancel, ok := s.cancels[jobID]; ok && cancel != nil {
		cancel()
	}
	job, exists := s.jobs[jobID]
	if !exists {
		s.mu.Unlock()
		return false
	}
	job.Status = transfer.StatusCancelled
	job.Phase = "cancelled"
	job.Error = "تم إلغاء عملية النسخ"
	cp := *job
	s.mu.Unlock()
	s.events.publish(transfer.TransferEvent{Type: transfer.EventJob, Job: &cp})
	return true
}

func (s *Service) SubscribeEvents(buffer int) (<-chan transfer.TransferEvent, func()) {
	return s.events.subscribe(buffer)
}

func (s *Service) updateJob(jobID string, update func(*transfer.TransferJob)) {
	s.mu.Lock()
	job, exists := s.jobs[jobID]
	if !exists {
		s.mu.Unlock()
		return
	}
	update(job)
	cp := *job
	s.mu.Unlock()
	s.events.publish(transfer.TransferEvent{Type: transfer.EventJob, Job: &cp})
}

func (s *Service) publishJob(job *transfer.TransferJob) {
	cp := *job
	s.events.publish(transfer.TransferEvent{Type: transfer.EventJob, Job: &cp})
}

func (s *Service) failJob(jobID string, err error) {
	now := time.Now()
	message := err.Error()
	if errors.Is(err, context.Canceled) {
		message = "تم إلغاء عملية النسخ"
	}
	s.updateJob(jobID, func(job *transfer.TransferJob) {
		job.CompletedAt = &now
		job.SpeedBps = 0
		job.SpeedMBps = 0
		job.ETASeconds = 0
		if errors.Is(err, context.Canceled) {
			job.Status = transfer.StatusCancelled
			job.Phase = "cancelled"
		} else {
			job.Status = transfer.StatusFailed
			job.Phase = "failed"
		}
		job.Error = message
	})
}

func (s *Service) cancelJobTerminal(jobID string) {
	now := time.Now()
	s.updateJob(jobID, func(job *transfer.TransferJob) {
		job.Status = transfer.StatusCancelled
		job.Phase = "cancelled"
		job.Error = "تم إلغاء عملية النسخ"
		job.CompletedAt = &now
		job.SpeedBps = 0
		job.SpeedMBps = 0
		job.ETASeconds = 0
	})
}

func inferDeviceType(deviceID string) string {
	switch {
	case strings.HasPrefix(deviceID, "disk_"):
		return string(transfer.DeviceStorage)
	case strings.HasPrefix(deviceID, "ios_"):
		return string(transfer.DeviceIOS)
	default:
		return string(transfer.DeviceAndroid)
	}
}

// remoteTargetPath resolves the on-device destination for a streamed source.
// It returns the parent directory (remoteDir) and the full destination path
// (remotePath). Storage and iOS use full file paths; the Android backend
// derives the folder and target name from the same full path internally.
func remoteTargetPath(deviceType transfer.DeviceType, req CopyRequest, displayName string) (string, string) {
	displayName = safeStageName(displayName)
	if displayName == "" || displayName == "." || displayName == ".." {
		displayName = "file"
	}

	switch deviceType {
	case transfer.DeviceStorage:
		driveLetter := strings.TrimPrefix(req.DeviceID, "disk_")
		rootPath := driveLetter + ":\\"
		target := strings.TrimSpace(req.TargetFolder)
		if target == "" {
			target = "Movies"
		}
		remoteDir := filepath.Join(rootPath, target)
		if sf := safeRelativeSub(req.SubFolder); sf != "" {
			remoteDir = filepath.Join(remoteDir, sf)
		}
		return remoteDir, filepath.Join(remoteDir, displayName)
	case transfer.DeviceIOS:
		remoteDir := "/Documents"
		if sf := safeRelativeSub(req.SubFolder); sf != "" {
			remoteDir += "/" + sf
		}
		return remoteDir, remoteDir + "/" + displayName
	default:
		target := strings.TrimSpace(req.TargetFolder)
		if target == "" {
			target = "Download"
		}
		remoteDir := target
		if sf := safeRelativeSub(req.SubFolder); sf != "" {
			remoteDir += "/" + sf
		}
		return remoteDir, remoteDir + "/" + displayName
	}
}

// safeRelativeSub sanitizes a sub-folder string so it can never escape the
// destination root or introduce path separators. Empty, dot and dot-dot
// segments are dropped, illegal characters are replaced, and any leading or
// trailing slashes are trimmed.
func safeRelativeSub(value string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	normalized = strings.Trim(normalized, "/")
	if normalized == "" {
		return ""
	}
	parts := strings.Split(normalized, "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = safeStageName(part)
		if part == "" || part == "." || part == ".." {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "/")
}

func progressSpan(index int, sources []sourceSpec, completedBytes, totalBytes int64) (float64, float64) {
	if totalBytes > 0 && len(sources) > 0 {
		currentSize := maxInt64(sources[index].Size, 0)
		if currentSize > 0 {
			start := float64(completedBytes) / float64(totalBytes) * 100
			end := float64(completedBytes+currentSize) / float64(totalBytes) * 100
			return clampFloat(start, 0, 100), clampFloat(end, 0, 100)
		}
	}
	count := len(sources)
	if count <= 0 {
		return 0, 100
	}
	start := float64(index) / float64(count) * 100
	end := float64(index+1) / float64(count) * 100
	return start, end
}

func recomputeTotalBytes(sources []sourceSpec) int64 {
	var total int64
	for _, source := range sources {
		total += maxInt64(source.Size, 0)
	}
	return total
}

func primarySourceName(sources []sourceSpec) string {
	if len(sources) == 0 {
		return "transfer"
	}
	return displaySourceName(sources[0])
}

func primarySourcePath(sources []sourceSpec) string {
	if len(sources) == 0 {
		return ""
	}
	if strings.TrimSpace(sources[0].Local) != "" {
		return sources[0].Local
	}
	return sources[0].URL
}

func displaySourceName(source sourceSpec) string {
	if strings.TrimSpace(source.Name) != "" {
		return strings.TrimSpace(source.Name)
	}
	if strings.TrimSpace(source.Local) != "" {
		return fileNameFromAnyPath(source.Local)
	}
	if strings.TrimSpace(source.URL) != "" {
		parsed, err := url.Parse(source.URL)
		if err == nil {
			base := path.Base(parsed.Path)
			if base != "" && base != "." && base != "/" {
				return base
			}
		}
	}
	return fmt.Sprintf("file-%d", source.Index+1)
}

func fileNameFromAnyPath(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if normalized == "" {
		return ""
	}
	parts := strings.Split(normalized, "/")
	return parts[len(parts)-1]
}

func safeStageName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			builder.WriteRune('_')
		default:
			builder.WriteRune(r)
		}
	}
	clean := strings.TrimSpace(builder.String())
	if clean == "" {
		return ""
	}
	return clean
}

func sumSourceSizes(items []sourceSpec) int64 {
	var total int64
	for _, item := range items {
		total += maxInt64(item.Size, 0)
	}
	return total
}

func indexedValue(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return strings.TrimSpace(values[index])
}

func indexedSize(values []int64, index int) int64 {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

func indexedInt64(values []int64, index int) int64 {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func clampInt64(value, minValue, maxValue int64) int64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func cleanupTempRoot(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

type eventBroker struct {
	mu   sync.RWMutex
	subs map[chan transfer.TransferEvent]struct{}
}

func newEventBroker() *eventBroker {
	return &eventBroker{subs: make(map[chan transfer.TransferEvent]struct{})}
}

func (b *eventBroker) subscribe(buffer int) (<-chan transfer.TransferEvent, func()) {
	ch := make(chan transfer.TransferEvent, buffer)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		if _, ok := b.subs[ch]; ok {
			delete(b.subs, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
}

func (b *eventBroker) publish(event transfer.TransferEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- event:
		default:
		}
	}
}
