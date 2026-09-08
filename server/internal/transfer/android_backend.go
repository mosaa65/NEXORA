package transfer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// AndroidBackend copies files to an Android phone exposed as a Windows MTP
// device via PowerShell Shell.Application (system "Phone" shell folder). It
// reproduces the proven logic from service.go (copyToMTPDevice /
// waitForMTPFile / queryMTPFileSize) as a self-contained TransferBackend.
//
// MTP has no synchronous write: CopyHere(16) launches the copy and returns
// immediately, so Put re-runs the PowerShell copy then polls the target file
// size until it reaches the source size (see plan §22, §24).
type AndroidBackend struct {
	deviceName string
}

// NewAndroidBackend builds an Android MTP backend with no initial state; the
// device name is bound at Connect time from the destination.
func NewAndroidBackend() *AndroidBackend {
	return &AndroidBackend{}
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

// Mkdir creates the given remote folder path on the phone using the same
// PowerShell script shell logic as copyToMTPDevice.
func (b *AndroidBackend) Mkdir(ctx context.Context, remotePath string) error {
	if strings.Trim(remotePath, "/ ") == "" {
		return nil
	}
	parts := b.targetParts(remotePath)
	targetSpec := strings.ReplaceAll(strings.Join(parts, "|"), "'", "''")
	deviceEscaped := strings.ReplaceAll(b.deviceName, "'", "''")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17)
if (-not $myComp) { Write-Error 'تعذر الوصول إلى أجهزة الكمبيوتر'; exit 1 }
$deviceItem = $null
foreach ($item in $myComp.Items()) {
    if ($item.Name -eq '%s') { $deviceItem = $item; break }
}
if (-not $deviceItem) {
    foreach ($item in $myComp.Items()) {
        if ($item.Type -like '*Portable*' -or $item.Type -like '*هاتف*' -or $item.Type -like '*جهاز*' -or ($item.Path -and $item.Path.StartsWith("::"))) {
            $deviceItem = $item; break
        }
    }
}
if (-not $deviceItem) { Write-Error 'لم يتم العثور على الجهاز الموصول عبر USB'; exit 1 }
$storage = $deviceItem.GetFolder
$firstStorage = $null
if ($storage) { $firstStorage = $storage.Items() | Select-Object -First 1 }
if (-not $firstStorage) { Write-Error 'تعذر الوصول إلى مجلد التخزين بالهاتف'; exit 1 }
$folder = $firstStorage.GetFolder
foreach ($name in '%s'.Split('|')) {
    if (-not $name) { continue }
    $next = $null
    foreach ($item in $folder.Items()) {
        if ($item.Name -eq $name) { $next = $item.GetFolder; break }
    }
    if (-not $next) {
        $folder.NewFolder($name)
        Start-Sleep -Milliseconds 500
        foreach ($item in $folder.Items()) {
            if ($item.Name -eq $name) { $next = $item.GetFolder; break }
        }
    }
    if (-not $next) { Write-Error "تعذر إنشاء أو فتح مجلد الوجهة: $name"; exit 1 }
    $folder = $next
}
`, deviceEscaped, targetSpec)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil || powershellOutputHasError(out) {
		reason := strings.TrimSpace(string(out))
		if reason == "" && err != nil {
			reason = err.Error()
		}
		return fmt.Errorf("فشل إنشاء مجلد الوجهة على الهاتف: %s", reason)
	}
	return nil
}

// Put copies source into the MTP folder derived from destination and polls the
// remote file size until it matches (or exceeds) the source size.
func (b *AndroidBackend) Put(ctx context.Context, source, destination string, opts PutOptions) error {
	parts := b.targetParts(destination)
	targetSpec := strings.ReplaceAll(strings.Join(parts, "|"), "'", "''")
	fileName := pathLeaf(source)
	deviceEscaped := strings.ReplaceAll(b.deviceName, "'", "''")
	sourceEscaped := strings.ReplaceAll(source, "'", "''")

	wantSize := statSize(source)
	timeout := transferTimeout(wantSize)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17)
$deviceItem = $null
if ($myComp) {
    foreach ($item in $myComp.Items()) {
        if ($item.Name -eq '%s') { $deviceItem = $item; break }
    }
}
if (-not $deviceItem -and $myComp) {
    foreach ($item in $myComp.Items()) {
        if ($item.Type -like '*Portable*' -or $item.Type -like '*هاتف*' -or $item.Type -like '*جهاز*' -or ($item.Path -and $item.Path.StartsWith("::"))) {
            $deviceItem = $item; break
        }
    }
}
if (-not $deviceItem) { Write-Error 'لم يتم العثور على الجهاز الموصول عبر USB'; exit 1 }
$storage = $deviceItem.GetFolder
$firstStorage = $null
if ($storage) { $firstStorage = $storage.Items() | Select-Object -First 1 }
if (-not $firstStorage) { Write-Error 'تعذر الوصول إلى مجلد التخزين بالهاتف'; exit 1 }
$finalTarget = $firstStorage.GetFolder
foreach ($name in '%s'.Split('|')) {
    if (-not $name) { continue }
    $nextFolder = $null
    foreach ($item in $finalTarget.Items()) {
        if ($item.Name -eq $name) { $nextFolder = $item.GetFolder; break }
    }
    if (-not $nextFolder) {
        $finalTarget.NewFolder($name)
        Start-Sleep -Milliseconds 500
        foreach ($item in $finalTarget.Items()) {
            if ($item.Name -eq $name) { $nextFolder = $item.GetFolder; break }
        }
    }
    if (-not $nextFolder) { Write-Error "تعذر إنشاء أو فتح مجلد الوجهة: $name"; exit 1 }
    $finalTarget = $nextFolder
}
foreach ($item in $finalTarget.Items()) {
    if ($item.Name -eq '%s') {
        try { $item.InvokeVerb('delete') } catch {}
        Start-Sleep -Milliseconds 300
        break
    }
}
$finalTarget.CopyHere('%s', 16)
`, deviceEscaped, targetSpec, strings.ReplaceAll(fileName, "'", "''"), sourceEscaped)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil || powershellOutputHasError(out) {
		reason := strings.TrimSpace(string(out))
		if reason == "" && err != nil {
			reason = err.Error()
		}
		return fmt.Errorf("فشل بدء نقل USB إلى الهاتف: %s", reason)
	}

	return b.waitForMTPFile(ctx, parts, fileName, wantSize, timeout, opts.OnProgress)
}

// waitForMTPFile polls the target file size until it reaches the source size.
func (b *AndroidBackend) waitForMTPFile(ctx context.Context, targetParts []string, fileName string, wantSize int64, timeout time.Duration, onProgress func(int64)) error {
	startTime := time.Now()
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			size, exists, err := queryMTPFileSize(ctx, b.deviceName, targetParts, fileName)
			if err != nil && time.Since(startTime) > 8*time.Second {
				return fmt.Errorf("تعذر التحقق من الملف داخل الهاتف: %w", err)
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
					return fmt.Errorf("توقف نقل USB قبل اكتمال الملف: وصل %s من %s", formatBytes(size), formatBytes(wantSize))
				}
				return fmt.Errorf("لم يظهر الملف داخل الهاتف بعد بدء النسخ خلال %s. غالباً تم رفض العملية أو انقطع اتصال USB", timeout.Round(time.Second))
			}
		}
	}
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

func (b *AndroidBackend) List(ctx context.Context, remotePath string) ([]RemoteEntry, error) {
	return nil, fmt.Errorf("قائمة المجلدات عبر MTP غير مدعومة في هذا الإصدار")
}

func (b *AndroidBackend) Delete(ctx context.Context, remotePath string) error {
	return fmt.Errorf("الحذف عبر MTP غير مدعوم في هذا الإصدار")
}

func (b *AndroidBackend) Rename(ctx context.Context, oldPath, newPath string) error {
	return fmt.Errorf("إعادة التسمية عبر MTP غير مدعومة في هذا الإصدار")
}

var _ TransferBackend = (*AndroidBackend)(nil)
