package transfer

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// AndroidBackend copies files to an Android phone exposed as a Windows MTP
// device via PowerShell Shell.Application (system "Phone" shell folder). It
// reproduces the proven logic from service.go (copyToMTPDevice /
// waitForMTPFile / queryMTPFileSize) as a self-contained TransferBackend.
//
// MTP has no synchronous write: CopyHere(16) launches the copy and returns
// immediately, so Put re-runs the PowerShell copy then polls the target file
// size until it reaches the source size (see plan §22, §24).
//
// WHAT THIS BACKEND CANNOT DO OVER SHELL COM (recorded, not worked around):
//
//   - Stream without a temp file. CopyHere takes a filesystem path, so
//     PutStream must spool the reader to disk first, which writes a large file
//     twice. This needs WPD's IStream.
//   - Resume. CopyHere has no offset/append interface, so a partial transfer
//     cannot continue. This needs WPD.
//   - Reliable rename. Shell COM offers no atomic rename over MTP.
//
// See docs/agent/ANDROID.md §2 and docs/agent/TRANSFER_ENGINE.md §8.
type AndroidBackend struct {
	deviceName string
	// storageName is the MTP storage to write into (e.g. "Internal storage",
	// "SD card"). Empty means "the first storage", which is the historical
	// behaviour and stays the default so nothing changes for existing callers.
	storageName string
}

// Markers the PowerShell scripts print. Output is parsed by these markers
// rather than matched loosely, so a missing device and a missing file are
// distinguishable instead of both looking like an empty result.
const (
	mtpMarkerDeviceMissing  = "NEXORA_MISSING_DEVICE"
	mtpMarkerStorageMissing = "NEXORA_MISSING_STORAGE"
	mtpMarkerDirMissing     = "NEXORA_MISSING_DIR"
	mtpMarkerItemMissing    = "NEXORA_MISSING_FILE"
	mtpMarkerFreePrefix     = "NEXORA_FREE:"
	mtpMarkerListPrefix     = "NEXORA_ENTRY:"
	mtpMarkerOK             = "NEXORA_OK"
)

// NewAndroidBackend builds an Android MTP backend with no initial state; the
// device name is bound at Connect time from the destination.
func NewAndroidBackend() *AndroidBackend {
	return &AndroidBackend{}
}

// NewAndroidBackendForStorage builds a backend that writes into a named MTP
// storage rather than always the first one. Existing callers keep using
// NewAndroidBackend, so this addition changes nothing for them.
func NewAndroidBackendForStorage(storageName string) *AndroidBackend {
	return &AndroidBackend{storageName: storageName}
}

func (b *AndroidBackend) Connect(ctx context.Context, device Device, destination TransferDestination) error {
	deviceName := destination.DeviceID
	if strings.HasPrefix(deviceName, "mtp_") || strings.HasPrefix(deviceName, "ios_") {
		parts := strings.SplitN(deviceName, "_", 3)
		if len(parts) == 3 {
			deviceName = strings.ReplaceAll(parts[2], "_", " ")
		}
	}
	if device.Name != "" {
		deviceName = device.Name
	}
	b.deviceName = deviceName
	return nil
}

func (b *AndroidBackend) Close() error { return nil }

// targetParts converts a remote path like "Movies/Season 1" into MTP folder
// name segments used by the PowerShell scripts.
func (b *AndroidBackend) targetParts(remotePath string) []string {
	return splitRemoteDir(remotePath)
}

// storageSelector is the PowerShell snippet that resolves $folder to the
// storage this backend writes into. An empty storageName keeps the historical
// "first storage" behaviour byte-for-byte.
func (b *AndroidBackend) storageSelector() string {
	if strings.TrimSpace(b.storageName) == "" {
	return `$storage = $deviceItem.GetFolder
$firstStorage = $null
if ($storage) { $firstStorage = $storage.Items() | Select-Object -First 1 }
if (-not $firstStorage) { Write-Output '` + mtpMarkerStorageMissing + `'; exit 0 }
$folder = $firstStorage.GetFolder`
	}
	escaped := strings.ReplaceAll(b.storageName, "'", "''")
	return `$storage = $deviceItem.GetFolder
$folder = $null
if ($storage) {
    foreach ($candidate in $storage.Items()) {
        if ($candidate.Name -eq '` + escaped + `') { $folder = $candidate.GetFolder; break }
    }
}
if (-not $folder) { Write-Output '` + mtpMarkerStorageMissing + `'; exit 0 }`
}

// connectPreamble is the shared PowerShell prologue: create the Shell object,
// find the named device, then select the storage. It prints a marker and exits
// when the device is gone, which is how a disconnect becomes a classified
// error instead of a generic failure.
func (b *AndroidBackend) connectPreamble() string {
	deviceEscaped := strings.ReplaceAll(b.deviceName, "'", "''")
	return `$ErrorActionPreference = 'Stop'
$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17)
$deviceItem = $null
if ($myComp) {
    foreach ($item in $myComp.Items()) {
        if ($item.Name -eq '` + deviceEscaped + `') { $deviceItem = $item; break }
    }
}
if (-not $deviceItem -and $myComp) {
    foreach ($item in $myComp.Items()) {
        if ($item.Type -like '*Portable*' -or $item.Type -like '*هاتف*' -or $item.Type -like '*جهاز*' -or ($item.Path -and $item.Path.StartsWith("::"))) {
            $deviceItem = $item; break
        }
    }
}
if (-not $deviceItem) { Write-Output '` + mtpMarkerDeviceMissing + `'; exit 0 }
` + b.storageSelector() + "\n"
}

// powershellPath resolves the Windows PowerShell host by absolute path.
//
// Invoking "powershell" by bare name depends on the process PATH, and the bare
// name is not guaranteed to resolve: System32 is absent from PATH under a
// restricted service account, under some agent launchers, and in the shell this
// project is developed in. The failure is a confusing "executable file not
// found in %PATH%" from an otherwise healthy backend. Resolving the known
// absolute locations first removes that dependency.
//
// The bare name stays as the final fallback so a machine where PowerShell is
// genuinely on PATH behaves exactly as before.
func powershellPath() string {
	candidates := []string{
	filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
	`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
	}
	for _, candidate := range candidates {
	if candidate == "" {
	continue
	}
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
	return candidate
	}
	}
	return "powershell"
}

// encodePowerShellCommand encodes a script for PowerShell's -EncodedCommand.
//
// The script is sent as UTF-16LE base64 rather than through -Command, because
// -Command passes the text on the process command line where the encoding and
// length are outside this program's control. The Android scripts contain Arabic
// folder matching, and a script that long is where the command line starts to
// truncate or mis-decode, which PowerShell then reports as a syntax error in
// the middle of an otherwise correct script. Encoding removes both problems.
func encodePowerShellCommand(script string) string {
	runes := utf16.Encode([]rune(script))
	buf := make([]byte, 0, len(runes)*2)
	for _, r := range runes {
	buf = append(buf, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// runPowerShell executes a script and returns its trimmed output. Errors are
// classified rather than passed through raw.
func (b *AndroidBackend) runPowerShell(ctx context.Context, script string, timeout time.Duration) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := exec.CommandContext(runCtx, powershellPath(),
	"-NoProfile", "-NonInteractive", "-EncodedCommand", encodePowerShellCommand(script)).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
	if runCtx.Err() != nil && ctx.Err() == nil {
	return text, NewTransferError(CodeMTPTimeout, "انتهت مهلة استجابة الهاتف", true, false)
	}
	if text == "" {
	return text, NewTransferError(CodeMTPError, err.Error(), true, false)
	}
	}
	if powershellOutputHasError(out) {
	return text, classifyMTPOutput(text)
	}
	return text, nil
}

// classifyMTPOutput turns a script marker or a PowerShell error into a
// classified TransferError, so the UI gets a code instead of free text.
func classifyMTPOutput(text string) *TransferError {
	switch {
	case strings.Contains(text, mtpMarkerDeviceMissing):
	return NewTransferError(CodeDeviceNotFound, "لم يتم العثور على الجهاز الموصول عبر USB", false, true)
	case strings.Contains(text, mtpMarkerStorageMissing):
	return NewTransferError(CodeStorageNotAvailable, "تعذر الوصول إلى مجلد التخزين بالهاتف", true, false)
	case strings.Contains(text, mtpMarkerDirMissing):
	return NewTransferError(CodeDestinationNotFound, "مسار الوجهة غير موجود على الهاتف", false, false)
	case strings.Contains(text, mtpMarkerItemMissing):
	return NewTransferError(CodeDestinationNotFound, "الملف غير موجود على الهاتف", false, false)
	default:
	return NewTransferError(CodeMTPError, strings.TrimSpace(text), true, false)
	}
}

// folderWalkScript walks (and optionally creates) each folder segment.
//
// It replaces the historical "create then Start-Sleep 500ms then re-scan" with
// a bounded poll: a slow device is given time to expose the new folder, and a
// fast one is not delayed. The fixed sleep was a guess at index latency;
// polling measures it.
func (b *AndroidBackend) folderWalkScript(targetParts []string, createMissing bool) string {
	targetSpec := strings.ReplaceAll(strings.Join(targetParts, "|"), "'", "''")
	createClause := ""
	if createMissing {
	createClause = `        $folder.NewFolder($name)
        $deadline = (Get-Date).AddSeconds(8)
        while ((Get-Date) -lt $deadline) {
            foreach ($candidate in $folder.Items()) {
                if ($candidate.Name -eq $name) { $next = $candidate.GetFolder; break }
            }
            if ($next) { break }
            Start-Sleep -Milliseconds 200
        }
`
	}
	return `$targetSpec = '` + targetSpec + `'
if ($targetSpec -ne '') {
    foreach ($name in $targetSpec.Split('|')) {
        if (-not $name) { continue }
        $next = $null
        foreach ($item in $folder.Items()) {
            if ($item.Name -eq $name) { $next = $item.GetFolder; break }
        }
        if (-not $next) {
` + createClause + `        }
        if (-not $next) { Write-Output '` + mtpMarkerDirMissing + `'; exit 0 }
        $folder = $next
    }
}
`
}

// walkToFolder runs the folder traversal, optionally creating missing folders.
func (b *AndroidBackend) walkToFolder(ctx context.Context, targetParts []string, createMissing bool, timeout time.Duration) error {
	script := b.connectPreamble() + b.folderWalkScript(targetParts, createMissing) + `Write-Output '` + mtpMarkerOK + "`\n"
	text, err := b.runPowerShell(ctx, script, timeout)
	if err != nil {
	return err
	}
	if !strings.Contains(text, mtpMarkerOK) {
	return classifyMTPOutput(text)
	}
	return nil
}

// FreeSpace reports the writable space of the selected storage, in bytes.
//
// It reads the Shell folder's space property, which is the same path the
// transfer uses, so the check cannot disagree with the copy.
func (b *AndroidBackend) FreeSpace(ctx context.Context) (int64, error) {
	script := b.connectPreamble() + `$free = $null
try { $free = $folder.ExtendedProperty('System.FreeSpace') } catch {}
if (-not $free) { try { $free = $folder.ExtendedProperty('System.StorageFreeSpace') } catch {} }
if (-not $free) { Write-Output '` + mtpMarkerStorageMissing + `'; exit 0 }
Write-Output ('` + mtpMarkerFreePrefix + `' + $free)
`
	text, err := b.runPowerShell(ctx, script, 20*time.Second)
	if err != nil {
	return 0, err
	}
	idx := strings.LastIndex(text, mtpMarkerFreePrefix)
	if idx < 0 {
	return 0, classifyMTPOutput(text)
	}
	raw := strings.TrimSpace(text[idx+len(mtpMarkerFreePrefix):])
	value, parseErr := parseHumanOrNumericSize(raw)
	if parseErr != nil {
	return 0, NewTransferError(CodeMTPError, "تعذر قراءة المساحة المتاحة على الهاتف", true, false)
	}
	return value, nil
}

// devicePresent reports whether the device is still attached. It is the cheap
// liveness probe the poll loop uses so a disconnect fails immediately instead
// of waiting out the whole transfer timeout.
func (b *AndroidBackend) devicePresent(ctx context.Context) bool {
	deviceEscaped := strings.ReplaceAll(b.deviceName, "'", "''")
	script := `$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17)
$found = $false
if ($myComp) {
    foreach ($item in $myComp.Items()) {
        if ($item.Name -eq '` + deviceEscaped + `') { $found = $true; break }
    }
}
if ($found) { Write-Output '` + mtpMarkerOK + `' }
`
	text, err := b.runPowerShell(ctx, script, 8*time.Second)
	if err != nil {
	return false
	}
	return strings.Contains(text, mtpMarkerOK)
}

// Mkdir creates the given remote folder path on the phone, creating any
// missing parent along the way.
func (b *AndroidBackend) Mkdir(ctx context.Context, remotePath string) error {
	if strings.Trim(remotePath, "/ ") == "" {
	return nil
	}
	return b.walkToFolder(ctx, b.targetParts(remotePath), true, 30*time.Second)
}

// Put copies source into MTP folder derived from destination and polls the
// remote file size until it matches (or exceeds) the source size.
func (b *AndroidBackend) Put(ctx context.Context, source, destination string, opts PutOptions) error {
	parts := b.targetParts(destination)
	fileName := leafName(source)
	wantSize := statSize(source)
	if wantSize <= 0 {
	return NewTransferError(CodeSourceNotFound, "الملف المصدر غير موجود أو فارغ", false, false)
	}

	// Fail before copying, not after, when the phone cannot hold the file.
	if free, err := b.FreeSpace(ctx); err == nil && free > 0 && wantSize > free {
	return NewTransferError(CodeInsufficientSpace,
	fmt.Sprintf("المساحة غير كافية على الهاتف: المتاح %s والمطلوب %s", formatBytes(free), formatBytes(wantSize)),
	false, false)
	}

	timeout := transferTimeout(wantSize)
	if err := b.walkToFolder(ctx, parts, true, timeout); err != nil {
	return err
	}

	sourceEscaped := strings.ReplaceAll(source, "'", "''")
	fileEscaped := strings.ReplaceAll(fileName, "'", "''")
	script := b.connectPreamble() + b.folderWalkScript(parts, true) + `foreach ($item in $folder.Items()) {
    if ($item.Name -eq '` + fileEscaped + `') {
        try { $item.InvokeVerb('delete') } catch {}
        Start-Sleep -Milliseconds 300
        break
    }
}
$folder.CopyHere('` + sourceEscaped + `', 16)
Write-Output '` + mtpMarkerOK + "`\n"

	if _, err := b.runPowerShell(ctx, script, timeout); err != nil {
	return err
	}

	return b.waitForMTPFile(ctx, parts, fileName, wantSize, timeout, opts.OnProgress)
}

// waitForMTPFile polls the target file size until it reaches the source size.
//
// CopyHere reports nothing, so this poll is the only completion signal MTP
// offers. It is a recorded limitation, not a design choice: a measured progress
// stream needs WPD (docs/agent/TRANSFER_ENGINE.md §8).
func (b *AndroidBackend) waitForMTPFile(ctx context.Context, targetParts []string, fileName string, wantSize int64, timeout time.Duration, onProgress func(int64)) error {
	startTime := time.Now()
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()

	for {
	select {
	case <-ctx.Done():
	return NewTransferError(CodeUserCancelled, "تم إلغاء النقل", false, false)
	case <-ticker.C:
		size, exists, err := queryMTPFileSize(ctx, b.deviceName, targetParts, fileName)
	if err != nil && time.Since(startTime) > 8*time.Second {
	// A device pulled mid-copy stops answering. Report that
	// specifically instead of a generic verification failure.
	return NewTransferError(CodeDeviceDisconnected,
	"تعذر التحقق من الملف داخل الهاتف — قد يكون الجهاز فُصل", true, true)
	}
	if exists {
	if onProgress != nil {
		onProgress(size)
	}
	if size >= wantSize {
	return nil
	}
	}
	if time.Since(startTime) > timeout {
	if exists {
	return NewTransferError(CodeTransferFailed,
	fmt.Sprintf("توقف نقل USB قبل اكتمال الملف: وصل %s من %s", formatBytes(size), formatBytes(wantSize)),
		true, false)
	}
	return NewTransferError(CodeTransferFailed,
	fmt.Sprintf("لم يظهر الملف داخل الهاتف بعد بدء النسخ خلال %s. غالباً تم رفض العملية أو انقطع اتصال USB", timeout.Round(time.Second)),
		true, false)
	}
	// Cheap liveness probe: a device that is gone can never finish the
	// copy, so fail now rather than after the full timeout.
	if !b.devicePresent(ctx) {
	return NewTransferError(CodeDeviceDisconnected, "تم فصل الجهاز أثناء النقل", true, true)
	}
	}
	}
}

// PutStream is not directly supported by MTP Shell COM without a temp file.
// We implement it by writing the reader to a temp file first, as MTP Shell API
// requires a source file path for CopyHere. The spool file is named after the
// destination's base name so the file lands on the phone with the correct
// title, and only the folder part of the destination is passed to Put (the
// Android backend derives the target name from the source file's leaf).
func (b *AndroidBackend) PutStream(ctx context.Context, reader io.Reader, size int64, destination string, opts PutOptions) error {
	dir, base := androidDestinationParts(destination)

	tempFile, err := os.CreateTemp("", "nexora-mtp-stream-*.bin")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if _, err := io.Copy(tempFile, reader); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	// Rename the spool to the destination's base name so MTP preserves it.
	spoolPath := filepath.Join(filepath.Dir(tempPath), base)
	if spoolPath != tempPath {
		if err := os.Remove(spoolPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(tempPath, spoolPath); err == nil {
			tempPath = spoolPath
		}
	}

	return b.Put(ctx, tempPath, dir, opts)
}

// androidDestinationParts splits a full destination path into the MTP folder
// (dir) and the desired file name (base). The Android backend always treats the
// folder part as the MTP target and derives the file name from the source, so
// this normalizes both Windows and forward-slash paths.
func androidDestinationParts(destination string) (dir, base string) {
	normalized := strings.ReplaceAll(destination, "\\", "/")
	hasTrailingSlash := strings.HasSuffix(normalized, "/")
	dir = path.Dir(normalized)
	base = leafName(normalized)
	if hasTrailingSlash || base == "" || base == "." || base == "/" || base == ".." {
		base = "stream-file.bin"
	}
	if dir == "." || dir == "" {
		dir = ""
	}
	return dir, base
}

func (b *AndroidBackend) Stat(ctx context.Context, remotePath string) (RemoteEntry, error) {
	parts := b.targetParts(remotePath)
	fileName := pathLeaf(remotePath)
	if len(parts) == 0 {
		return RemoteEntry{}, nil
	}
	size, exists, err := queryMTPFileSize(ctx, b.deviceName, parts, fileName)
	if err != nil {
		return RemoteEntry{}, err
	}
	if !exists {
		return RemoteEntry{}, fmt.Errorf("الملف غير موجود على الجهاز: %s", remotePath)
	}
	return RemoteEntry{Name: fileName, Path: remotePath, Size: size}, nil
}

// List returns the entries directly under remotePath on the phone.
//
// Each entry carries an explicit type and size, so the parser builds
// RemoteEntry values from the data rather than inferring them from a display
// name.
func (b *AndroidBackend) List(ctx context.Context, remotePath string) ([]RemoteEntry, error) {
	parts := b.targetParts(remotePath)
	script := b.connectPreamble() + b.folderWalkScript(parts, false) + `foreach ($item in $folder.Items()) {
    $isFolder = $false
    if ($item.IsFolder) { $isFolder = $true }
    $size = 0
    if (-not $isFolder) {
        $size = $item.ExtendedProperty('System.Size')
        if (-not $size) { $size = $item.Size }
        if (-not $size) { $size = 0 }
    }
    $kind = 'F'
    if ($isFolder) { $kind = 'D' }
    Write-Output ('` + mtpMarkerListPrefix + `' + $kind + '|' + $size + '|' + $item.Name)
}
`
	text, err := b.runPowerShell(ctx, script, 60*time.Second)
	if err != nil {
	return nil, err
	}
	// A folder that does not exist yet is an empty listing, not an error, so
	// browsing "Movies" before the first copy is not a failure.
	if strings.Contains(text, mtpMarkerDirMissing) {
	return []RemoteEntry{}, nil
	}
	if strings.Contains(text, mtpMarkerDeviceMissing) || strings.Contains(text, mtpMarkerStorageMissing) {
	return nil, classifyMTPOutput(text)
	}
	return parseMTPListing(text, remotePath), nil
}

// parseMTPListing turns the marked script output into RemoteEntry values.
//
// A line that is not a well-formed entry is ignored rather than fabricated, so
// a stray warning cannot become a phantom file in the listing.
func parseMTPListing(text, remotePath string) []RemoteEntry {
	entries := make([]RemoteEntry, 0, 16)
	for _, line := range strings.Split(text, "\n") {
	line = strings.TrimSpace(line)
	idx := strings.Index(line, mtpMarkerListPrefix)
	if idx < 0 {
	continue
	}
	payload := line[idx+len(mtpMarkerListPrefix):]
	fields := strings.SplitN(payload, "|", 3)
	if len(fields) != 3 {
	continue
	}
	name := fields[2]
	if name == "" {
	continue
	}
	size, sizeErr := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
	if sizeErr != nil {
	size = 0
	}
	entryPath := name
	if strings.Trim(remotePath, "/ ") != "" {
	entryPath = strings.TrimRight(remotePath, "/") + "/" + name
	}
	entries = append(entries, RemoteEntry{
	Name:  name,
	Path:  entryPath,
	IsDir: fields[0] == "D",
	Size:  size,
	})
	}
	return entries
}

// Delete removes a remote file or folder by invoking the shell item's delete
// verb — the same mechanism Put already uses to replace an existing file, so
// this exposes no capability the backend did not already exercise.
func (b *AndroidBackend) Delete(ctx context.Context, remotePath string) error {
	parts := b.targetParts(remotePath)
	name := pathLeaf(remotePath)
	if name == "" {
	return NewTransferError(CodeDestinationNotFound, "مسار الحذف فارغ", false, false)
	}
	escaped := strings.ReplaceAll(name, "'", "''")
	parent := parts
	if len(parent) > 0 {
	parent = parent[:len(parent)-1]
	}
	script := b.connectPreamble() + b.folderWalkScript(parent, false) + `$target = $null
foreach ($item in $folder.Items()) {
    if ($item.Name -eq '` + escaped + `') { $target = $item; break }
}
if (-not $target) { Write-Output '` + mtpMarkerItemMissing + `'; exit 0 }
try { $target.InvokeVerb('delete') } catch { Write-Output '` + mtpMarkerItemMissing + `'; exit 0 }
Write-Output '` + mtpMarkerOK + "`\n"
	_, err := b.runPowerShell(ctx, script, 30*time.Second)
	return err
}

// Rename is not implemented over Shell COM, and is reported as such rather
// than approximated.
//
// Shell COM offers no atomic rename over MTP: there is no rename verb, and a
// copy-then-delete would need a second full write of the file. A classified
// UNSUPPORTED_OPERATION lets a caller fall back deliberately instead of
// silently corrupting a name. Moving this to WPD is recorded in
// docs/agent/ROADMAP.md.
func (b *AndroidBackend) Rename(ctx context.Context, oldPath, newPath string) error {
	return NewTransferError(CodeUnsupportedOperation,
	"إعادة التسمية على أجهزة MTP غير مدعومة عبر Shell COM — تحتاج WPD", false, false)
}

var _ TransferBackend = (*AndroidBackend)(nil)
