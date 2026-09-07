package transfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	goios "github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/afc"
	"github.com/danielpaulus/go-ios/ios/house_arrest"
)

// GoIOSBackend transfers files through go-ios AFC/House Arrest connections.
// It keeps one connection for the selected device and application.
type GoIOSBackend struct {
	deviceID string
	bundleID string
	client   *afc.Client
}

func NewGoIOSBackend() *GoIOSBackend { return &GoIOSBackend{} }
func goIOSDeviceEntry(deviceID string) (goios.DeviceEntry, error) {
	udid, ok := parseIOSUDID(deviceID)
	if !ok {
		return goios.DeviceEntry{}, fmt.Errorf("معرف الآيفون غير صالح: %s", deviceID)
	}
	devices, err := goios.ListDevices()
	if err != nil {
		return goios.DeviceEntry{}, fmt.Errorf("تعذر اكتشاف أجهزة iOS: %w", err)
	}
	for _, entry := range devices.DeviceList {
		if entry.Properties.SerialNumber == udid {
			return entry, nil
		}
	}
	return goios.DeviceEntry{}, fmt.Errorf("جهاز iOS غير متصل أو غير موثوق: %s", udid)
}

func (b *GoIOSBackend) Connect(ctx context.Context, device Device, destination TransferDestination) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	udid, ok := parseIOSUDID(destination.DeviceID)
	if !ok {
		return fmt.Errorf("معرف الآيفون غير صالح: %s", destination.DeviceID)
	}
	if strings.TrimSpace(destination.AppID) == "" {
		return fmt.Errorf("معرف تطبيق iOS مطلوب")
	}

	entry, err := goIOSDeviceEntry(destination.DeviceID)
	if err != nil {
		return err
	}

	client, err := house_arrest.New(entry, destination.AppID)
	if err != nil {
		return fmt.Errorf("تعذر فتح Documents للتطبيق %s: %w", destination.AppID, err)
	}
	b.deviceID = udid
	b.bundleID = destination.AppID
	b.client = client
	return nil
}

func (b *GoIOSBackend) requireClient() (*afc.Client, error) {
	if b.client == nil {
		return nil, fmt.Errorf("اتصال iOS غير مهيأ")
	}
	return b.client, nil
}

func (b *GoIOSBackend) List(ctx context.Context, remoteDir string) ([]RemoteEntry, error) {
	client, err := b.requireClient()
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	names, err := client.List(remoteDir)
	if err != nil {
		return nil, err
	}
	entries := make([]RemoteEntry, 0, len(names))
	for _, name := range names {
		info, statErr := client.Stat(path.Join(remoteDir, name))
		if statErr != nil {
			return nil, statErr
		}
		entries = append(entries, RemoteEntry{
			Name: name, Path: path.Join(remoteDir, name), IsDir: info.IsDir(), Size: info.Size,
		})
	}
	return entries, nil
}

func (b *GoIOSBackend) Stat(ctx context.Context, remotePath string) (RemoteEntry, error) {
	client, err := b.requireClient()
	if err != nil {
		return RemoteEntry{}, err
	}
	select {
	case <-ctx.Done():
		return RemoteEntry{}, ctx.Err()
	default:
	}
	info, err := client.Stat(remotePath)
	if err != nil {
		return RemoteEntry{}, err
	}
	return RemoteEntry{Name: path.Base(remotePath), Path: remotePath, IsDir: info.IsDir(), Size: info.Size}, nil
}

func (b *GoIOSBackend) Mkdir(ctx context.Context, remoteDir string) error {
	client, err := b.requireClient()
	if err != nil {
		return err
	}
	current := ""
	for _, part := range splitRemoteDir(remoteDir) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		current += "/" + part
		if err := client.MkDir(current); err != nil {
			if _, statErr := client.Stat(current); statErr != nil {
				return err
			}
		}
	}
	return nil
}

func (b *GoIOSBackend) Put(ctx context.Context, source, destination string, opts PutOptions) error {
	client, err := b.requireClient()
	if err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	mode := afc.WRITE_ONLY_CREATE_TRUNC
	transferred := int64(0)
	if opts.ResumeOffset > 0 {
		mode = afc.WRITE_ONLY_CREATE_APPEND
		if _, err := in.Seek(opts.ResumeOffset, io.SeekStart); err != nil {
			return err
		}
		transferred = opts.ResumeOffset
	}
	out, err := client.Open(destination, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if opts.OnProgress != nil && transferred > 0 {
		opts.OnProgress(transferred)
	}

	bufferSize := int(opts.BufferSize)
	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}
	if bufferSize > 512*1024 {
		bufferSize = 512 * 1024
	}
	buffer := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, readErr := in.Read(buffer)
		if n > 0 {
			written, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			transferred += int64(written)
			if opts.OnProgress != nil {
				opts.OnProgress(transferred)
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func iosDocumentsPath(input string) string {
	clean := strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "Documents" {
		return "/Documents"
	}
	clean = strings.TrimPrefix(clean, "Documents/")
	return "/Documents/" + safeRelativePath(clean)
}

func (b *GoIOSBackend) Delete(ctx context.Context, remotePath string) error {
	client, err := b.requireClient()
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return client.Remove(remotePath)
}

func (b *GoIOSBackend) Rename(ctx context.Context, oldPath, newPath string) error {
	return fmt.Errorf("go-ios AFC لا يوفر إعادة تسمية مباشرة: %s -> %s", oldPath, newPath)
}

func (b *GoIOSBackend) Close() error {
	if b.client == nil {
		return nil
	}
	err := b.client.Close()
	b.client = nil
	return err
}

var _ TransferBackend = (*GoIOSBackend)(nil)
