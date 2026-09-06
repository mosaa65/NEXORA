package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	goios "github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/installationproxy"
)

type Service struct {
	options Options
	mu      sync.RWMutex
	jobs    map[string]*TransferJob
	history []*TransferJob

	engine *TransferEngine

	cachedDevices []Device
	lastScan      time.Time
	stopMonitor   chan struct{}

	events *eventBroker
}

func NewService(opts Options) *Service {
	if opts.AndroidTargetFolder == "" {
		opts.AndroidTargetFolder = "Download"
	}
	if opts.IOSBundleID == "" {
		opts.IOSBundleID = "org.videolan.vlc-ios"
	}
	s := &Service{
		options:       opts,
		jobs:          make(map[string]*TransferJob),
		history:       make([]*TransferJob, 0, 50),
		engine:        newV2Engine(opts),
		cachedDevices: make([]Device, 0, 4),
		stopMonitor:   make(chan struct{}),
		events:        newEventBroker(),
	}
	// Live progress: the engine reports every job transition, the service
	// mirrors it onto the legacy REST job and fans it out to SSE subscribers.
	s.engine.SetNotifier(func(v2 *TransferJobV2) {
		s.mu.RLock()
		job, ok := s.jobs[v2.ID]
		s.mu.RUnlock()
		if !ok {
			return
		}
		s.mirrorV2Progress(job, v2)
	})
	go s.startDeviceMonitor()
	return s
}

func (s *Service) Close() {
	if s.stopMonitor != nil {
		select {
		case <-s.stopMonitor:
		default:
			close(s.stopMonitor)
		}
	}
}

func (s *Service) startDeviceMonitor() {
	// Immediate first scan
	s.refreshDevicesBackground()

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopMonitor:
			return
		case <-ticker.C:
			s.refreshDevicesBackground()
		}
	}
}

func (s *Service) refreshDevicesBackground() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	devices, err := s.scanDevicesInternal(ctx)
	if err == nil {
		s.mu.Lock()
		changed := !devicesEqual(s.cachedDevices, devices)
		s.cachedDevices = devices
		s.lastScan = time.Now()
		s.mu.Unlock()
		if changed {
			cp := append([]Device(nil), devices...)
			s.events.publish(TransferEvent{Type: EventDevices, Devices: cp})
		}
	}
}

// devicesEqual compares two device lists field-by-field (order matters).
func devicesEqual(a, b []Device) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// snapshotDevices returns a copy of the cached device list.
func (s *Service) snapshotDevices() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Device(nil), s.cachedDevices...)
}

// SnapshotDevices returns a copy of the cached device list (exported for the
// API layer to seed SSE clients instantly).
func (s *Service) SnapshotDevices() []Device {
	return s.snapshotDevices()
}

// newV2Engine builds the v2 transfer engine whose backend factory selects the
// concrete transfer backend by device type (see plan §71-§72).
func newV2Engine(opts Options) *TransferEngine {
	eng := NewTransferEngine(DefaultTransferConfig(), func(device Device) (TransferBackend, error) {
		switch device.Type {
		case DeviceIOS:
			if isIOSUDID(device.ID) {
				return NewGoIOSBackend(), nil
			}
			return NewAndroidBackend(), nil
		case DeviceStorage:
			return NewStorageBackend(), nil
		default:
			return NewAndroidBackend(), nil
		}
	})
	return eng
}

// ListDevices returns connected USB Android (MTP) and iOS devices instantly from hot cache.
func (s *Service) ListDevices(ctx context.Context) ([]Device, error) {
	s.mu.RLock()
	cached := s.cachedDevices
	last := s.lastScan
	s.mu.RUnlock()

	// Return cached data immediately if less than 4 seconds old
	if len(cached) > 0 && time.Since(last) < 4*time.Second {
		return cached, nil
	}

	devices, err := s.scanDevicesInternal(ctx)
	if err == nil {
		s.mu.Lock()
		s.cachedDevices = devices
		s.lastScan = time.Now()
		s.mu.Unlock()
		return devices, nil
	}

	if len(cached) > 0 {
		return cached, nil
	}
	return devices, err
}

func (s *Service) scanDevicesInternal(ctx context.Context) ([]Device, error) {
	devices := make([]Device, 0, 4)

	// 1. iOS device discovery via go-ios/usbmuxd
	iosCtx, iosCancel := context.WithTimeout(ctx, 2*time.Second)
	iosDevices, err := s.discoverIOSDevices(iosCtx)
	iosCancel()
	if err == nil && len(iosDevices) > 0 {
		devices = append(devices, iosDevices...)
	}

	// 2. Windows Portable Devices / MTP / Apple iPhone Discovery
	if runtime.GOOS == "windows" {
		mtpCtx, mtpCancel := context.WithTimeout(ctx, 2500*time.Millisecond)
		androidDevices, err := s.discoverWindowsMTPDevices(mtpCtx)
		mtpCancel()
		if err == nil {
			if len(iosDevices) > 0 {
				androidDevices = filterWindowsIOSPlaceholders(androidDevices)
			}
			devices = append(devices, androidDevices...)
		}
	}

	// 3. Removable USB flash drives (always discovered alongside phones)
	if runtime.GOOS == "windows" {
		removable, _ := s.discoverRemovableDrives(ctx)
		if len(removable) > 0 {
			devices = append(devices, removable...)
		}
	}

	return devices, nil
}

func filterWindowsIOSPlaceholders(devices []Device) []Device {
	filtered := devices[:0]
	for _, device := range devices {
		if isWindowsIOSPlaceholder(device) {
			continue
		}
		filtered = append(filtered, device)
	}
	return filtered
}

func isWindowsIOSPlaceholder(device Device) bool {
	name := strings.ToLower(device.Name)
	return strings.HasPrefix(device.ID, "ios_mtp_") ||
		device.Type == DeviceIOS ||
		strings.Contains(name, "iphone") ||
		strings.Contains(name, "ipad") ||
		strings.Contains(name, "apple")
}

func (s *Service) discoverWindowsMTPDevices(ctx context.Context) ([]Device, error) {
	// Query Shell Portable Devices AND PnP Connected Mobile Devices (Android & iPhone)
	psCmd := `
$devices = @()
$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17) # 17 = ssfDRIVES (My Computer)

if ($myComp) {
    foreach ($item in $myComp.Items()) {
        $n = $item.Name
        $t = $item.Type
        if ($n -and ($t -like "*Portable*" -or $t -like "*Camera*" -or $t -like "*هاتف*" -or $t -like "*جهاز*" -or $n -like "*iPhone*" -or $n -like "*Apple*" -or $n -like "*iPad*" -or $n -like "*Android*" -or $n -like "*Galaxy*" -or $n -like "*Huawei*" -or $n -like "*Redmi*" -or $n -like "*Xiaomi*" -or ($item.Path -and $item.Path.StartsWith("::")))) {
            $devType = "android"
            if ($n -like "*iPhone*" -or $n -like "*Apple*" -or $n -like "*iPad*") {
                $devType = "ios"
            }
            $free = 0
            $total = 0
            try {
                $folder = $item.GetFolder
                if ($folder) {
                    foreach ($sub in $folder.Items()) {
                        $f = $sub.ExtendedProperty("System.FreeSpace")
                        $c = $sub.ExtendedProperty("System.Capacity")
                        if ($f -and [int64]$f -gt 0) { $free += [int64]$f }
                        if ($c -and [int64]$c -gt 0) { $total += [int64]$c }
                    }
                }
            } catch {}
            $devices += [PSCustomObject]@{
                ID = $n
                Name = $n
                Type = $devType
                Status = "متصل"
                FreeSpace = $free
                TotalSpace = $total
            }
        }
    }
}

# Fallback: Check connected PnP Mobile Devices if Shell didn't return any
if ($devices.Count -eq 0) {
    try {
        $pnp = Get-PnpDevice -ErrorAction SilentlyContinue | Where-Object { $_.Status -eq 'OK' -and ($_.FriendlyName -like '*iPhone*' -or $_.FriendlyName -like '*Apple*' -or $_.FriendlyName -like '*iPad*' -or $_.FriendlyName -like '*Android*' -or $_.FriendlyName -like '*Galaxy*' -or $_.FriendlyName -like '*Huawei*' -or $_.FriendlyName -like '*Redmi*' -or $_.FriendlyName -like '*Xiaomi*') }
        foreach ($dev in $pnp) {
            $devType = "android"
            if ($dev.FriendlyName -like "*iPhone*" -or $dev.FriendlyName -like "*Apple*" -or $dev.FriendlyName -like "*iPad*") {
                $devType = "ios"
            }
            $devices += [PSCustomObject]@{
                ID = $dev.InstanceId
                Name = $dev.FriendlyName
                Type = $devType
                Status = "متصل"
                FreeSpace = 0
                TotalSpace = 0
            }
        }
    } catch {}
}

$devices | ConvertTo-Json -Compress
`
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var parsed []windowsPortableRecord
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			return buildWindowsPortableDevices(parsed), nil
		}
	} else {
		var one windowsPortableRecord
		if err := json.Unmarshal([]byte(raw), &one); err == nil {
			return buildWindowsPortableDevices([]windowsPortableRecord{one}), nil
		}
	}

	return nil, nil
}

func buildWindowsPortableDevices(records []windowsPortableRecord) []Device {
	list := make([]Device, 0, len(records))
	for i, item := range records {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		deviceType, model, prefix, status := windowsPortableDeviceKind(name, item.Type)
		list = append(list, Device{
			ID:         fmt.Sprintf("%s%d_%s", prefix, i, name),
			Name:       name,
			Model:      model,
			Type:       deviceType,
			Status:     status,
			FreeSpace:  item.FreeSpace,
			TotalSpace: item.TotalSpace,
		})
	}
	return list
}

func windowsPortableDeviceKind(name, dType string) (DeviceType, string, string, string) {
	nameLower := strings.ToLower(name)
	if dType == "ios" ||
		strings.Contains(nameLower, "iphone") ||
		strings.Contains(nameLower, "apple") ||
		strings.Contains(nameLower, "ipad") {
		return DeviceIOS, "هاتف آيفون", "ios_", "متصل"
	}
	return DeviceAndroid, "هاتف أندرويد", "mtp_", "متصل"
}

func (s *Service) discoverIOSDevices(ctx context.Context) ([]Device, error) {
	result := make(chan struct {
		devices goios.DeviceList
		err     error
	}, 1)
	go func() {
		devices, err := goios.ListDevices()
		result <- struct {
			devices goios.DeviceList
			err     error
		}{devices, err}
	}()
	var response struct {
		devices goios.DeviceList
		err     error
	}
	select {
	case response = <-result:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if response.err != nil {
		return nil, nil
	}

	list := make([]Device, 0, len(response.devices.DeviceList))
	for i, entry := range response.devices.DeviceList {
		udid := entry.Properties.SerialNumber
		if udid == "" {
			continue
		}
		var freeBytes, totalBytes int64
		if lockdown, err := goios.ConnectLockdownWithSession(entry); err == nil {
			if values, err := lockdown.GetValues(); err == nil {
				if strings.TrimSpace(values.Value.DeviceName) != "" {
					name = strings.TrimSpace(values.Value.DeviceName)
				}
				if val, err := lockdown.GetValue("com.apple.disk_usage", "TotalDataAvailable"); err == nil {
					if n, ok := val.(uint64); ok {
						freeBytes = int64(n)
					} else if n, ok := val.(int64); ok {
						freeBytes = n
					}
				}
				if val, err := lockdown.GetValue("com.apple.disk_usage", "TotalDiskCapacity"); err == nil {
					if n, ok := val.(uint64); ok {
						totalBytes = int64(n)
					} else if n, ok := val.(int64); ok {
						totalBytes = n
					}
				}
			}
			lockdown.Close()
		}
		list = append(list, Device{
			ID:         fmt.Sprintf("ios_%d_%s", i, udid),
			Name:       name,
			Model:      "هاتف آيفون",
			Type:       DeviceIOS,
			Status:     "متصل",
			FreeSpace:  freeBytes,
			TotalSpace: totalBytes,
		})
	}
	return list, nil
}

func (s *Service) ListDeviceApps(ctx context.Context, deviceID string) ([]DeviceApp, error) {
	entry, err := goIOSDeviceEntry(deviceID)
	if err != nil {
		return nil, err
	}
	proxy, err := installationproxy.New(entry)
	if err != nil {
		return nil, fmt.Errorf("تعذر الاتصال بخدمة التطبيقات: %w", err)
	}
	defer proxy.Close()
	// go-ios v1.3.2's BrowseFileSharingApps request does not set the
	// ApplicationType filter. Read the complete plist and use Apple's explicit
	// UIFileSharingEnabled flag instead.
	installed, err := proxy.BrowseAllApps()
	if err != nil {
		return nil, fmt.Errorf("تعذر قراءة تطبيقات مشاركة الملفات: %w", err)
	}
	apps := make([]DeviceApp, 0, len(installed))
	for _, app := range installed {
		fileSharing, ok := app[installationproxy.UIFileSharingEnabled].(bool)
		if !ok || !fileSharing {
			continue
		}
		bundleID := app.CFBundleIdentifier()
		if bundleID == "" {
			continue
		}
		name := bundleID
		if value, ok := app[installationproxy.CFBundleDisplayName].(string); ok && value != "" {
			name = value
		} else if value, ok := app[installationproxy.CFBundleName].(string); ok && value != "" {
			name = value
		}
		apps = append(apps, DeviceApp{BundleID: bundleID, Name: name, Icon: iosAppIcon(name, bundleID), Category: "تطبيق مثبت على الآيفون", FileSharingEnabled: true})
	}
	return apps, nil
}

func (s *Service) ListAppFolders(ctx context.Context, deviceID, bundleID string) ([]AppFolder, error) {
	if _, ok := parseIOSUDID(deviceID); !ok {
		return nil, fmt.Errorf("معرف الآيفون غير صالح")
	}
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		bundleID = s.options.IOSBundleID
	}
	if bundleID == "" {
		bundleID = "org.videolan.vlc-ios"
	}

	backend := NewGoIOSBackend()
	defer backend.Close()
	if err := backend.Connect(ctx, Device{ID: deviceID, Type: DeviceIOS}, TransferDestination{DeviceID: deviceID, DeviceType: DeviceIOS, AppID: bundleID}); err != nil {
		return nil, err
	}
	entries, err := backend.List(ctx, "/Documents")
	if err != nil {
		return nil, err
	}
	folders := make([]AppFolder, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			folders = append(folders, AppFolder{Name: entry.Name, Path: strings.TrimPrefix(entry.Path, "/Documents/"), IsDir: true})
		}
	}
	return folders, nil
}


func (s *Service) StartCopy(ctx context.Context, req CopyRequest) (*TransferJob, error) {
	paths := req.sourcePaths()
	if len(paths) == 0 {
		return nil, errors.New("مسار الملف المصدر مطلوب")
	}

	var totalSize int64
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			return nil, errors.New("مسار الملف المصدر مطلوب")
		}
		fileInfo, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("فشل القراءة من الملف المصدر (%s): %w", p, err)
		}
		if fileInfo.IsDir() {
			return nil, errors.New("المسار المحدد مجلد وليس ملف فيديو: " + p)
		}
		totalSize += fileInfo.Size()
	}

	dev, found := s.findDevice(req.DeviceID)
	deviceName := req.DeviceID
	if found && dev.Name != "" {
		deviceName = dev.Name
	}

	// 1. Pre-check: Free disk space
	if found && dev.FreeSpace > 0 && totalSize > dev.FreeSpace {
		return nil, fmt.Errorf("المساحة المتوفرة على القرص (%s) غير كافية لحجم الملفات المطلوب (%s)", formatBytes(dev.FreeSpace), formatBytes(totalSize))
	}

	// 2. Pre-check: FAT32 4GB limit per file
	if found && strings.EqualFold(dev.FileSystem, "FAT32") {
		for _, p := range paths {
			if fi, err := os.Stat(p); err == nil && fi.Size() >= 4294967295 {
				return nil, fmt.Errorf("نظام ملفات القرص (FAT32) لا يدعم ملفات أكبر من 4 جيجابايت. الملف '%s' بحجم %s يتجاوز هذا الحد. يرجى تهيئة الفلاشة بنظام NTFS أو exFAT", filepath.Base(p), formatBytes(fi.Size()))
			}
		}
	}

	primary := paths[0]
	jobID := fmt.Sprintf("job_%d_%s", time.Now().UnixNano(), filepath.Base(primary))

	jobCtx, cancel := context.WithCancel(context.Background())

	job := &TransferJob{
		ID:          jobID,
		DeviceID:    req.DeviceID,
		DeviceName:  deviceName,
		DeviceType:  "android",
		SourcePath:  primary,
		FileName:    filepath.Base(primary),
		FileSize:    totalSize,
		FileCount:   int64(len(paths)),
		Transferred: 0,
		Progress:    0.0,
		Phase:       "queued",
		Status:      StatusPending,
		StartedAt:   time.Now(),
		cancelFunc:  cancel,
	}

	if isIOSUDID(req.DeviceID) {
		job.DeviceType = "ios"
	} else if strings.HasPrefix(req.DeviceID, "disk_") {
		job.DeviceType = "storage"
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.history = append([]*TransferJob{job}, s.history...)
	if len(s.history) > 100 {
		s.history = s.history[:100]
	}
	s.mu.Unlock()

	go s.executeTransferMulti(jobCtx, job, req, paths)

	return job, nil
}

// sourcePaths resolves the list of source files for a CopyRequest, supporting
// both the legacy single source_path and the Phase-4 source_paths[] array.
func (req CopyRequest) sourcePaths() []string {
	var out []string
	for _, p := range req.SourcePaths {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) > 0 {
		return out
	}
	if p := strings.TrimSpace(req.SourcePath); p != "" {
		out = append(out, p)
	}
	return out
}

// executeTransferMulti drives a (possibly multi-file) copy request to a
// terminal state, mirroring the outcome onto the legacy job.
func (s *Service) executeTransferMulti(ctx context.Context, job *TransferJob, req CopyRequest, paths []string) {
	s.updateJob(job.ID, func(j *TransferJob) {
		j.Status = StatusProcessing
		j.Phase = "preparing"
		if j.Progress <= 0 {
			j.Progress = 1
		}
	})

	err := s.runTransferV2(ctx, job, req, paths)

	now := time.Now()
	s.updateJob(job.ID, func(j *TransferJob) {
		j.CompletedAt = &now
		if err != nil {
			if errors.Is(err, context.Canceled) {
				j.Status = StatusCancelled
				j.Error = "تم إلغاء عملية النسخ"
				j.Phase = "cancelled"
			} else if errors.Is(err, context.DeadlineExceeded) {
				j.Status = StatusFailed
				j.Error = "انتهت مهلة النسخ قبل اكتمال الملف. افحص الكابل وافتح قفل الهاتف ثم حاول مرة أخرى"
				j.Phase = "failed"
			} else {
				j.Status = StatusFailed
				j.Error = err.Error()
				j.Phase = "failed"
			}
		} else {
			j.Status = StatusCompleted
			j.Progress = 100.0
			j.Transferred = j.FileSize
			j.FinishedFiles = int64(len(paths))
			j.Phase = "completed"
		}
	})
}

// runTransferV2 drives the copy through the v2 engine and the concrete
// backends, then mirrors the final outcome onto the legacy job. The proven
// copyTo* helpers remain as a documented fallback (see copyToMTPDevice etc.)
// if a v2 destination cannot be described.
func (s *Service) runTransferV2(ctx context.Context, job *TransferJob, req CopyRequest, paths []string) error {
	files := s.filesFor(paths)
	if len(files) == 0 {
		return s.executeTransferLegacy(ctx, job, req, paths)
	}
	v2 := s.buildV2Job(req, files)
	if v2 == nil {
		return s.executeTransferLegacy(ctx, job, req, paths)
	}

	// For the engine notifier to find this job in s.jobs, the v2 job must
	// carry the same id as the legacy REST job it mirrors.
	v2.ID = job.ID

	// Route the copy through the engine.
	if err := s.engine.Submit(v2); err != nil {
		// Fall back to the legacy path if the engine refuses the job.
		return s.executeTransferLegacy(ctx, job, req, paths)
	}

	// Publish the queued state immediately; the engine notifier mirrors all
	// subsequent transitions onto the legacy job and into the event broker.
	if snapshot, ok := s.engine.GetJob(v2.ID); ok {
		s.mirrorV2Progress(job, snapshot)
	}

	// Wait for the terminal job event instead of polling the engine registry.
	stream, unsub := s.events.subscribe(256)
	defer unsub()
	for {
		select {
		case <-ctx.Done():
			s.engine.cancelJob(v2.ID)
			return ctx.Err()
		case ev := <-stream:
			if ev.Type != EventJob || ev.Job == nil || ev.Job.ID != job.ID {
				continue
			}
			if legacyIsTerminal(ev.Job) {
				return legacyTerminalOutcome(ev.Job)
			}
		}
	}
}

// filesFor converts source paths into TransferFile entries with stat sizes.
func (s *Service) filesFor(paths []string) []TransferFile {
	if len(paths) == 0 {
		return nil
	}
	files := make([]TransferFile, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var size int64
		if info, err := os.Stat(p); err == nil {
			size = info.Size()
		}
		files = append(files, TransferFile{SourcePath: p, Size: size})
	}
	return files
}

// buildV2Job converts a CopyRequest and its resolved files into a v2 multi-file
// engine job. It returns nil when the request cannot be described by
// TransferDestination (in which case the legacy path is used).
func (s *Service) buildV2Job(req CopyRequest, files []TransferFile) *TransferJobV2 {
	if strings.TrimSpace(req.DeviceID) == "" {
		return nil
	}
	if len(files) == 0 {
		files = s.filesFor(req.sourcePaths())
	}
	if len(files) == 0 {
		return nil
	}

	var dest TransferDestination
	switch {
	case strings.HasPrefix(req.DeviceID, "disk_"):
		driveLetter := strings.TrimPrefix(req.DeviceID, "disk_")
		rootPath := driveLetter + ":\\"
		target := strings.TrimSpace(req.TargetFolder)
		remote := rootPath
		if target != "" && target != "/" && target != "\\Documents" && target != "/Documents" {
			if strings.Contains(target, ":") {
				remote = target
			} else {
				remote = filepath.Join(rootPath, target)
			}
		}
		if sf := safeRelativePath(req.SubFolder); sf != "" {
			remote = filepath.Join(remote, sf)
		}
		dest = TransferDestination{
			DeviceID:   req.DeviceID,
			DeviceType: DeviceStorage,
			RemotePath: remote,
		}
	case isIOSUDID(req.DeviceID):
		bundleID := firstNonEmpty(req.TargetApp, s.options.IOSBundleID, "org.videolan.vlc-ios")
		remote := iosDocumentsPath(req.SubFolder)
		dest = TransferDestination{
			DeviceID:   req.DeviceID,
			DeviceType: DeviceIOS,
			AppID:      bundleID,
			RemotePath: remote,
		}
	case strings.HasPrefix(req.DeviceID, "ios_"):
		target := safeRelativePath(req.TargetFolder)
		if target == "" {
			target = firstNonEmpty(req.TargetApp, "Movies")
		}
		remote := target
		if sf := safeRelativePath(req.SubFolder); sf != "" {
			remote += "/" + sf
		}
		dest = TransferDestination{
			DeviceID:   req.DeviceID,
			DeviceType: DeviceIOS,
			AppID:      req.TargetApp,
			RemotePath: remote,
		}
	default:
		target := safeRelativePath(req.TargetFolder)
		if target == "" {
			target = firstNonEmpty(s.options.AndroidTargetFolder, "Movies")
		}
		remote := target
		if sf := safeRelativePath(req.SubFolder); sf != "" {
			remote += "/" + sf
		}
		dest = TransferDestination{
			DeviceID:   req.DeviceID,
			DeviceType: DeviceAndroid,
			RemotePath: remote,
		}
	}

	var total int64
	for _, f := range files {
		total += f.Size
	}
	if total == 0 {
		total = 1
	}

	return &TransferJobV2{
		ID:          jobID(req.SourcePath),
		DeviceID:    req.DeviceID,
		DeviceType:  dest.DeviceType,
		Destination: dest,
		Files:       files,
		TotalBytes:  total,
		Status:      StatusQueued,
	}
}

// mirrorV2Progress copies live progress fields from a v2 job onto the legacy
// job used by the REST layer.
func (s *Service) mirrorV2Progress(job *TransferJob, v2 *TransferJobV2) {
	s.mu.Lock()
	job.Progress = v2.Progress
	job.Transferred = v2.TransferredBytes
	job.TransferredBytes = v2.TransferredBytes
	job.TotalBytes = v2.TotalBytes
	job.Phase = string(v2.Phase)
	job.Status = v2.Status
	job.CurrentFile = v2.CurrentFile
	if v2.CurrentFile != "" {
		job.FileName = v2.CurrentFile
	}
	job.SpeedBps = v2.SpeedBps
	job.SpeedMBps = float64(v2.SpeedBps) / (1024 * 1024)
	job.ETASeconds = int64(v2.ETA.Seconds())
	if v2.Destination.RemotePath != "" {
		job.DestinationPath = v2.Destination.RemotePath
	}
	if v2.Status == StatusFailed && v2.Error != nil {
		job.Error = fmt.Sprintf("%s: %s", v2.Error.Code, v2.Error.Message)
	}
	if len(v2.Files) > 0 {
		job.FileSize = v2.TotalBytes
		job.FileCount = int64(len(v2.Files))
		job.TotalFiles = int64(len(v2.Files))
		finished := 0
		for _, f := range v2.Files {
			if f.Status == StatusCompleted {
				finished++
			}
		}
		job.FinishedFiles = int64(finished)
		job.CompletedFiles = int64(finished)
	}
	cp := *job
	s.mu.Unlock()

	s.events.publish(TransferEvent{Type: EventJob, Job: &cp})
}

// SubscribeEvents returns a channel of live transfer events (job progress and
// device-set changes) plus a cancel function. Any payload type may arrive.
func (s *Service) SubscribeEvents(buffer int) (<-chan TransferEvent, func()) {
	return s.events.subscribe(buffer)
}

// legacyIsTerminal reports whether a legacy job reached a final state.
func legacyIsTerminal(j *TransferJob) bool {
	return j.Status == StatusCompleted || j.Status == StatusFailed || j.Status == StatusCancelled
}

// legacyTerminalOutcome derives the error value for a terminal legacy job, so
// the finaliser reports a delay that matches its status.
func legacyTerminalOutcome(j *TransferJob) error {
	switch j.Status {
	case StatusCompleted:
		return nil
	case StatusCancelled:
		return context.Canceled
	default:
		if j.Error != "" {
			return errors.New(j.Error)
		}
		return errors.New("فشل النقل")
	}
}

// v2Outcome converts a terminal v2 job into the error value expected by the
// legacy job finaliser.
func v2Outcome(v2 *TransferJobV2) error {
	switch v2.Status {
	case StatusCompleted:
		return nil
	case StatusCancelled:
		return context.Canceled
	default:
		if v2.Error != nil {
			return fmt.Errorf("%s: %s", v2.Error.Code, v2.Error.Message)
		}
		return fmt.Errorf("فشل النقل (%s)", v2.Status)
	}
}

// browserBackendFor builds a v2 destination from a device ID plus an optional
// iOS app bundle id and returns a ready-to-use backend for File Browser tasks
// (Phase 3).
func (s *Service) browserBackendFor(ctx context.Context, deviceID, appID string, deviceType DeviceType) (TransferBackend, TransferDestination, error) {
	dest := TransferDestination{DeviceID: deviceID, DeviceType: deviceType, AppID: appID}
	backend, err := s.engine.backendFor(dest)
	if err != nil {
		return nil, dest, err
	}
	return backend, dest, nil
}

// ListDevicePath lists the entries directly under a remote folder (Phase 3).
func (s *Service) ListDevicePath(ctx context.Context, deviceID, path string, deviceType DeviceType, appID string) ([]RemoteEntry, error) {
	if deviceType == "" {
		deviceType = deviceTypeOf(deviceID)
	}
	backend, _, err := s.browserBackendFor(ctx, deviceID, appID, deviceType)
	if err != nil {
		return nil, err
	}
	defer backend.Close()
	if err := backend.Connect(ctx, Device{ID: deviceID, Type: deviceType}, TransferDestination{DeviceID: deviceID, DeviceType: deviceType, AppID: appID}); err != nil {
		return nil, err
	}
	return backend.List(ctx, path)
}

// StatDevicePath returns metadata for a single remote path (Phase 3).
func (s *Service) StatDevicePath(ctx context.Context, deviceID, path string, deviceType DeviceType, appID string) (RemoteEntry, error) {
	if deviceType == "" {
		deviceType = deviceTypeOf(deviceID)
	}
	backend, _, err := s.browserBackendFor(ctx, deviceID, appID, deviceType)
	if err != nil {
		return RemoteEntry{}, err
	}
	defer backend.Close()
	if err := backend.Connect(ctx, Device{ID: deviceID, Type: deviceType}, TransferDestination{DeviceID: deviceID, DeviceType: deviceType, AppID: appID}); err != nil {
		return RemoteEntry{}, err
	}
	return backend.Stat(ctx, path)
}

// CreateDeviceFolder creates a directory (and missing parents) on a device
// (Phase 3).
func (s *Service) CreateDeviceFolder(ctx context.Context, deviceID, path string, deviceType DeviceType, appID string) error {
	if deviceType == "" {
		deviceType = deviceTypeOf(deviceID)
	}
	backend, _, err := s.browserBackendFor(ctx, deviceID, appID, deviceType)
	if err != nil {
		return err
	}
	defer backend.Close()
	if err := backend.Connect(ctx, Device{ID: deviceID, Type: deviceType}, TransferDestination{DeviceID: deviceID, DeviceType: deviceType, AppID: appID}); err != nil {
		return err
	}
	return backend.Mkdir(ctx, path)
}

// deviceTypeOf resolves a DeviceType from a device ID, duplicating the legacy
// classification used when a CopyRequest creates its job.
func deviceTypeOf(deviceID string) DeviceType {
	if isIOSUDID(deviceID) {
		return DeviceIOS
	}
	if strings.HasPrefix(deviceID, "disk_") {
		return DeviceStorage
	}
	return DeviceAndroid
}

// executeTransferLegacy is the proven per-device dispatch that predates the v2
// engine. It is kept as a documented fallback when a v2 destination cannot be
// described or the engine refuses a job. For a multi-file request it copies the
// files one at a time (the v2 engine is preferred for true parallel multi-file).
func (s *Service) executeTransferLegacy(ctx context.Context, job *TransferJob, req CopyRequest, paths []string) error {
	if len(paths) == 0 {
		paths = []string{req.SourcePath}
	}
	for i, src := range paths {
		if errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		err := s.copyOneLegacy(ctx, job, req, src, i, len(paths))
		if err != nil {
			return err
		}
	}
	return nil
}

// copyOneLegacy copies a single source file through the proven per-device
// helper, temporarily repointing the legacy job's current-file fields.
func (s *Service) copyOneLegacy(ctx context.Context, job *TransferJob, req CopyRequest, src string, index, total int) error {
	fileInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("فشل القراءة من الملف المصدر (%s): %w", src, err)
	}
	origSource, origName, origSize := job.SourcePath, job.FileName, job.FileSize
	job.SourcePath = src
	job.FileName = filepath.Base(src)
	job.FileSize = fileInfo.Size()
	defer func() {
		job.SourcePath = origSource
		job.FileName = origName
		if total > 1 {
			job.FileSize = origSize
			job.FinishedFiles = int64(index + 1)
		} else {
			job.FileSize = origSize
		}
	}()

	if strings.HasPrefix(req.DeviceID, "disk_") {
		driveLetter := strings.TrimPrefix(req.DeviceID, "disk_") + ":\\"
		target := strings.TrimSpace(req.TargetFolder)
		targetFolder := driveLetter
		if target != "" && target != "/" && target != "\\Documents" && target != "/Documents" {
			if strings.Contains(target, ":") {
				targetFolder = target
			} else {
				targetFolder = filepath.Join(driveLetter, target)
			}
		}
		if sf := safeRelativePath(req.SubFolder); sf != "" {
			targetFolder = filepath.Join(targetFolder, sf)
		}
		return s.copyToLocalPath(ctx, job, src, targetFolder)
	}
	if isIOSUDID(req.DeviceID) {
		return s.copyToIOSDeviceGoIOS(ctx, job, req)
	}
	return s.copyToMTPDevice(ctx, job, req)
}

// jobID generates a stable unique job identifier from a source path.
func jobID(sourcePath string) string {
	return fmt.Sprintf("job_%d_%s", time.Now().UnixNano(), filepath.Base(sourcePath))
}

func (s *Service) copyToIOSDeviceGoIOS(ctx context.Context, job *TransferJob, req CopyRequest) error {
	bundleID := firstNonEmpty(req.TargetApp, s.options.IOSBundleID, "org.videolan.vlc-ios")
	remoteDir := iosDocumentsPath(req.SubFolder)
	remotePath := remoteDir + "/" + safePathComponent(job.FileName)
	displayPath := fmt.Sprintf("آيفون > %s > %s", bundleID, remotePath)
	s.updateJob(job.ID, func(j *TransferJob) {
		j.DestinationPath = displayPath
		j.Phase = "copying"
		if j.Progress < 1 {
			j.Progress = 1
		}
	})

	backend := NewGoIOSBackend()
	defer backend.Close()
	if err := backend.Connect(ctx, Device{ID: req.DeviceID, Type: DeviceIOS}, TransferDestination{DeviceID: req.DeviceID, DeviceType: DeviceIOS, AppID: bundleID}); err != nil {
		return err
	}
	if err := backend.Mkdir(ctx, remoteDir); err != nil {
		return err
	}
	start := time.Now()
	if err := backend.Put(ctx, req.SourcePath, remotePath, PutOptions{
		BufferSize:   defaultBufferSize,
		ResumeOffset: remoteResumeOffset(ctx, backend, remotePath, job.FileSize),
		OnProgress:   func(transferred int64) { s.updateProgress(job, transferred, start) },
	}); err != nil {
		return err
	}
	s.updateJob(job.ID, func(j *TransferJob) { j.Phase = "verifying" })
	entry, err := backend.Stat(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("تعذر التحقق من الملف على الآيفون: %w", err)
	}
	if entry.Size != job.FileSize {
		return fmt.Errorf("حجم الملف على الآيفون (%s) لا يطابق المصدر (%s)", formatBytes(entry.Size), formatBytes(job.FileSize))
	}
	return nil
}

func (s *Service) copyToMTPDevice(ctx context.Context, job *TransferJob, req CopyRequest) error {
	deviceName := req.DeviceID
	if strings.HasPrefix(deviceName, "mtp_") || strings.HasPrefix(deviceName, "ios_mtp_") {
		parts := strings.SplitN(deviceName, "_", 3)
		if len(parts) == 3 {
			deviceName = parts[2]
		}
	}

	sourceEscaped := strings.ReplaceAll(req.SourcePath, "'", "''")
	targetFolder := safeRelativePath(req.TargetFolder)
	if targetFolder == "" {
		targetFolder = s.options.AndroidTargetFolder
	}
	if targetFolder == "" {
		targetFolder = "Movies"
	}

	subFolder := safeRelativePath(req.SubFolder)
	targetParts := append(splitRelativePath(targetFolder), splitRelativePath(subFolder)...)
	targetSpec := strings.ReplaceAll(strings.Join(targetParts, "|"), "'", "''")

	destPathReadable := fmt.Sprintf("الهاتف > التخزين الداخلي > %s", targetFolder)
	if subFolder != "" {
		destPathReadable += fmt.Sprintf(" > %s", subFolder)
	}
	destPathReadable += fmt.Sprintf(" > %s", job.FileName)

	s.updateJob(job.ID, func(j *TransferJob) {
		j.DestinationPath = destPathReadable
		j.Phase = "copying"
		if j.Progress < 1 {
			j.Progress = 1
		}
	})

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17)
$deviceItem = $null

if ($myComp) {
    foreach ($item in $myComp.Items()) {
        if ($item.Name -eq '%s') {
            $deviceItem = $item
            break
        }
    }
}

if (-not $deviceItem -and $myComp) {
    foreach ($item in $myComp.Items()) {
        if ($item.Type -like '*Portable*' -or $item.Type -like '*هاتف*' -or $item.Type -like '*جهاز*' -or ($item.Path -and $item.Path.StartsWith("::"))) {
            $deviceItem = $item
            break
        }
    }
}

if (-not $deviceItem) {
    Write-Error "لم يتم العثور على الجهاز الموصول عبر USB"
    exit 1
}

$storage = $deviceItem.GetFolder
$firstStorage = $null
if ($storage) { $firstStorage = $storage.Items() | Select-Object -First 1 }

if (-not $firstStorage) {
    Write-Error "تعذر الوصول إلى مجلد التخزين بالهاتف"
    exit 1
}

$finalTarget = $firstStorage.GetFolder
$targetSpec = '%s'
if ($targetSpec -ne '') {
    foreach ($name in $targetSpec.Split('|')) {
        if (-not $name) { continue }
        $nextFolder = $null
        foreach ($item in $finalTarget.Items()) {
            if ($item.Name -eq $name) {
                $nextFolder = $item.GetFolder
                break
            }
        }
        if (-not $nextFolder) {
            $finalTarget.NewFolder($name)
            Start-Sleep -Milliseconds 500
            foreach ($item in $finalTarget.Items()) {
                if ($item.Name -eq $name) {
                    $nextFolder = $item.GetFolder
                    break
                }
            }
        }
        if (-not $nextFolder) {
            Write-Error "تعذر إنشاء أو فتح مجلد الوجهة: $name"
            exit 1
        }
        $finalTarget = $nextFolder
    }
}

foreach ($item in $finalTarget.Items()) {
    if ($item.Name -eq '%s') {
        try { $item.InvokeVerb('delete') } catch {}
        Start-Sleep -Milliseconds 300
        break
    }
}

$finalTarget.CopyHere('%s', 16)
`, deviceName, targetSpec, strings.ReplaceAll(job.FileName, "'", "''"), sourceEscaped)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil || powershellOutputHasError(out) {
		reason := strings.TrimSpace(string(out))
		if reason == "" && err != nil {
			reason = err.Error()
		}
		return fmt.Errorf("فشل بدء نقل USB إلى الهاتف: %s", reason)
	}

	return s.waitForMTPFile(ctx, job, deviceName, targetParts, job.FileName)
}

func (s *Service) waitForMTPFile(ctx context.Context, job *TransferJob, deviceName string, targetParts []string, fileName string) error {
	startTime := time.Now()
	timeout := transferTimeout(job.FileSize)
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()

	s.updateJob(job.ID, func(j *TransferJob) {
		j.Phase = "verifying"
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			size, exists, err := queryMTPFileSize(ctx, deviceName, targetParts, fileName)
			if err != nil && time.Since(startTime) > 8*time.Second {
				return fmt.Errorf("تعذر التحقق من الملف داخل الهاتف: %w", err)
			}
			if exists {
				s.updateProgress(job, size, startTime)
				if size >= job.FileSize {
					return nil
				}
			}
			if time.Since(startTime) > timeout {
				if exists {
					return fmt.Errorf("توقف نقل USB قبل اكتمال الملف: وصل %s من %s", formatBytes(size), formatBytes(job.FileSize))
				}
				return fmt.Errorf("لم يظهر الملف داخل الهاتف بعد بدء النسخ خلال %s. غالباً تم رفض العملية أو انقطع اتصال USB", timeout.Round(time.Second))
			}
		}
	}
}

func queryMTPFileSize(ctx context.Context, deviceName string, targetParts []string, fileName string) (int64, bool, error) {
	deviceEscaped := strings.ReplaceAll(deviceName, "'", "''")
	fileEscaped := strings.ReplaceAll(fileName, "'", "''")
	targetSpec := strings.ReplaceAll(strings.Join(targetParts, "|"), "'", "''")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17)
$deviceItem = $null
foreach ($item in $myComp.Items()) {
    if ($item.Name -eq '%s') { $deviceItem = $item; break }
}
if (-not $deviceItem) { Write-Output 'NEXORA_MISSING_DEVICE'; exit 0 }
$storage = $deviceItem.GetFolder
$firstStorage = $storage.Items() | Select-Object -First 1
if (-not $firstStorage) { Write-Output 'NEXORA_MISSING_STORAGE'; exit 0 }
$folder = $firstStorage.GetFolder
$targetSpec = '%s'
if ($targetSpec -ne '') {
    foreach ($name in $targetSpec.Split('|')) {
        if (-not $name) { continue }
        $found = $null
        foreach ($item in $folder.Items()) {
            if ($item.Name -eq $name) { $found = $item.GetFolder; break }
        }
        if (-not $found) { Write-Output 'NEXORA_MISSING_FILE'; exit 0 }
        $folder = $found
    }
}
foreach ($item in $folder.Items()) {
    if ($item.Name -eq '%s') {
        $size = $item.ExtendedProperty('System.Size')
        if (-not $size) { $size = $item.Size }
        Write-Output "NEXORA_SIZE:$size"
        exit 0
    }
}
Write-Output 'NEXORA_MISSING_FILE'
`, deviceEscaped, targetSpec, fileEscaped)

	pollCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	out, err := exec.CommandContext(pollCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return 0, false, err
	}
	text := strings.TrimSpace(string(out))
	if strings.Contains(text, "NEXORA_SIZE:") {
		raw := strings.TrimSpace(text[strings.LastIndex(text, "NEXORA_SIZE:")+len("NEXORA_SIZE:"):])
		size, err := parseHumanOrNumericSize(raw)
		if err != nil {
			return 0, true, nil
		}
		return size, true, nil
	}
	return 0, false, nil
}

func (s *Service) copyToIOSDevice(ctx context.Context, job *TransferJob, req CopyRequest) error {
	return s.copyToIOSDeviceGoIOS(ctx, job, req)
}

func (s *Service) copyToLocalPath(ctx context.Context, job *TransferJob, source, targetFolder string) error {
	if err := os.MkdirAll(targetFolder, 0o755); err != nil {
		return fmt.Errorf("فشل إنشاء مجلد الوجهة: %w", err)
	}

	targetPath := filepath.Join(targetFolder, filepath.Base(source))
	s.updateJob(job.ID, func(j *TransferJob) {
		j.DestinationPath = targetPath
		j.Phase = "copying"
		if j.Progress < 1 {
			j.Progress = 1
		}
	})
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 1024*1024)
	var transferred int64
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			_ = out.Close()
			_ = os.Remove(targetPath)
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

			elapsed := time.Since(startTime).Seconds()
			speedMBps := 0.0
			if elapsed > 0 {
				speedMBps = (float64(transferred) / (1024 * 1024)) / elapsed
			}
			progress := (float64(transferred) / float64(job.FileSize)) * 100.0

			s.updateJob(job.ID, func(j *TransferJob) {
				j.Transferred = transferred
				j.Progress = progress
				j.SpeedMBps = speedMBps
			})
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return readErr
		}
	}

	return nil
}

func (s *Service) updateProgress(job *TransferJob, transferred int64, startTime time.Time) {
	if transferred < 0 {
		transferred = 0
	}
	if job.FileSize > 0 && transferred > job.FileSize {
		transferred = job.FileSize
	}
	elapsed := time.Since(startTime).Seconds()
	speedMBps := 0.0
	if elapsed > 0 {
		speedMBps = (float64(transferred) / (1024 * 1024)) / elapsed
	}
	progress := 1.0
	if job.FileSize > 0 {
		progress = (float64(transferred) / float64(job.FileSize)) * 100.0
		if progress < 1 && transferred > 0 {
			progress = 1
		}
		if progress > 99.5 && transferred < job.FileSize {
			progress = 99.5
		}
	}

	s.updateJob(job.ID, func(j *TransferJob) {
		j.Transferred = transferred
		j.Progress = progress
		j.SpeedMBps = speedMBps
	})
}

func (s *Service) GetJob(jobID string) (*TransferJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.jobs[jobID]
	if !exists {
		return nil, false
	}
	// return copy
	cp := *job
	return &cp, true
}

func (s *Service) ListJobs() []*TransferJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*TransferJob, len(s.history))
	for i, j := range s.history {
		cp := *j
		result[i] = &cp
	}
	return result
}

func (s *Service) CancelJob(jobID string) bool {
	s.mu.Lock()
	job, exists := s.jobs[jobID]
	if !exists {
		s.mu.Unlock()
		return false
	}
	if job.cancelFunc != nil {
		job.cancelFunc()
	}
	job.Status = StatusCancelled
	cp := *job
	s.mu.Unlock()

	s.events.publish(TransferEvent{Type: EventJob, Job: &cp})
	return true
}

func (s *Service) updateJob(jobID string, fn func(j *TransferJob)) {
	s.mu.Lock()
	job, exists := s.jobs[jobID]
	if !exists {
		s.mu.Unlock()
		return
	}
	fn(job)
	cp := *job
	s.mu.Unlock()

	s.events.publish(TransferEvent{Type: EventJob, Job: &cp})
}

func extractJSONValue(str, key string) string {
	idx := strings.Index(str, `"`+key+`"`)
	if idx == -1 {
		return ""
	}
	sub := str[idx+len(key)+3:]
	colonIdx := strings.Index(sub, ":")
	if colonIdx == -1 {
		return ""
	}
	valPart := strings.TrimSpace(sub[colonIdx+1:])
	valPart = strings.TrimPrefix(valPart, `"`)
	endQuote := strings.Index(valPart, `"`)
	if endQuote != -1 {
		return valPart[:endQuote]
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isIOSUDID(deviceID string) bool {
	_, ok := parseIOSUDID(deviceID)
	return ok
}

func parseIOSUDID(deviceID string) (string, bool) {
	if strings.HasPrefix(deviceID, "ios_mtp_") {
		return "", false
	}
	if strings.HasPrefix(deviceID, "ios_") {
		parts := strings.SplitN(deviceID, "_", 3)
		if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
			udid := strings.TrimSpace(parts[2])
			if isLikelyAppleUDID(udid) {
				return udid, true
			}
		}
		return "", false
	}
	deviceID = strings.TrimSpace(deviceID)
	if !isLikelyAppleUDID(deviceID) {
		return "", false
	}
	return deviceID, true
}

func isLikelyAppleUDID(value string) bool {
	if len(value) < 24 {
		return false
	}
	hasHex := false
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			hasHex = true
		case r >= 'a' && r <= 'f':
			hasHex = true
		case r >= 'A' && r <= 'F':
			hasHex = true
		case r == '-':
		default:
			return false
		}
	}
	return hasHex
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func iosAppIcon(name, bundleID string) string {
	value := strings.ToLower(name + " " + bundleID)
	switch {
	case strings.Contains(value, "vlc"):
		return "🟧"
	case strings.Contains(value, "infuse"):
		return "🔴"
	case strings.Contains(value, "document") || strings.Contains(value, "readdle"):
		return "🟦"
	case strings.Contains(value, "player") || strings.Contains(value, "video"):
		return "🎬"
	default:
		return "📦"
	}
}

func powershellOutputHasError(output []byte) bool {
	text := strings.ToLower(string(output))
	return strings.Contains(text, "write-error") ||
		strings.Contains(text, "fullyqualifiederrorid") ||
		strings.Contains(text, "exception calling") ||
		strings.Contains(text, "cannot find") ||
		strings.Contains(text, "access is denied")
}

func (s *Service) findDevice(id string) (Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.cachedDevices {
		if d.ID == id {
			return d, true
		}
	}
	return Device{}, false
}

// EjectDevice safely flushes buffers and requests Windows shell to eject the USB drive.
func (s *Service) EjectDevice(ctx context.Context, deviceID string) error {
	if !strings.HasPrefix(deviceID, "disk_") {
		return errors.New("الإخراج الآمن مدعوم فقط لوحدات تخزين USB")
	}
	letter := strings.TrimPrefix(deviceID, "disk_")
	if len(letter) == 0 {
		return errors.New("حرف القرص غير صالح")
	}
	drivePath := fmt.Sprintf("%s:", strings.ToUpper(letter))
	if runtime.GOOS == "windows" {
		psCmd := fmt.Sprintf(`(New-Object -com Shell.Application).Namespace(17).ParseName('%s').InvokeVerb('Eject')`, drivePath)
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
		_ = cmd.Run()
		go s.refreshDevicesBackground()
		return nil
	}
	return nil
}
