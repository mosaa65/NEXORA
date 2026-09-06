// Command transfertest exercises the v2 transfer engine against a real
// connected device (Android MTP, iOS AFC, or a local disk drive) from the
// CLI, printing live progress. It uses the same public Service API as the
// REST layer (StartCopy / GetJob / CancelJob), so it validates the actual
// integration path — not a mock.
//
// Example (Android):
//
//	go run ./cmd/transfertest -device "mtp_0_SM-G991B" -src "D:\movies\a.mp4" -folder Movies -sub "Series 1"
//
// Example (iOS):
//
//	go run ./cmd/transfertest -device "ios_0_00008100-000123456789A1CE10203040506070" \
//	    -src "D:\movies\a.mp4" -app "org.videolan.vlc-ios" -sub "Series 1"
//
// Example (USB disk):
//
//	go run ./cmd/transfertest -device "disk_E" -src "D:\movies\a.mp4"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"nexora/server/internal/transfer"
)

func main() {
	var (
		device = flag.String("device", "", "device id: mtp_..., ios_..., disk_E")
		src    = flag.String("src", "", "absolute path to the source file (repeat -src to copy multiple files)")
		folder = flag.String("folder", "", "target folder on device (e.g. Movies)")
		sub    = flag.String("sub", "", "sub-folder under the target (e.g. Series 1)")
		app    = flag.String("app", "", "iOS bundle id (only for iOS)")
		list   = flag.Bool("list", false, "list connected devices and exit")
		browse = flag.String("browse", "", "list remote folder for device (Phase 3)")
		mkdir  = flag.String("mkdir", "", "create remote folder for device (Phase 3)")
	)
	flag.Parse()

	svc := transfer.NewService(transfer.Options{
		AndroidTargetFolder: *folder,
		IOSBundleID:         *app,
	})

	if *list {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		devices, err := svc.ListDevices(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ListDevices: %v\n", err)
			os.Exit(1)
		}
		if len(devices) == 0 {
			fmt.Println("no devices detected")
			return
		}
		for _, d := range devices {
			fmt.Printf("%-28s %-8s %s\n", d.ID, d.Type, d.Name)
		}
		return
	}

	pending := flag.Args()

	if *browse != "" || *mkdir != "" {
		if *device == "" {
			flag.Usage()
			os.Exit(2)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		if *mkdir != "" {
			if err := svc.CreateDeviceFolder(ctx, *device, *mkdir, "", *app); err != nil {
				fmt.Fprintf(os.Stderr, "CreateDeviceFolder: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("folder created: %s\n", *mkdir)
		}
		if *browse != "" {
			entries, err := svc.ListDevicePath(ctx, *device, *browse, "", *app)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ListDevicePath: %v\n", err)
				os.Exit(1)
			}
			if len(entries) == 0 {
				fmt.Println("(empty)")
			}
			for _, e := range entries {
				kind := "file"
				if e.IsDir {
					kind = "dir "
				}
				fmt.Printf("%-5s %-10s %s\n", kind, humanBytes(e.Size), e.Name)
			}
		}
		return
	}

	if *device == "" || (len(pending) == 0 && *src == "") {
		flag.Usage()
		os.Exit(2)
	}

	sources := []string{}
	if *src != "" {
		sources = append(sources, *src)
	}
	sources = append(sources, pending...)

	req := transfer.CopyRequest{
		DeviceID:     *device,
		SourcePath:   sources[0],
		SourcePaths:  sources,
		TargetApp:    *app,
		SubFolder:    *sub,
		TargetFolder: *folder,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	job, err := svc.StartCopy(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "StartCopy: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("job=%s device=%s phase=%s\n", job.ID, job.DeviceID, job.Phase)

	// Poll until terminal. Ctrl-C calls CancelJob via the notify context only if
	// the engine notices; we also cancel directly on signal by watching ctx.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastPct float64
	for {
		select {
		case <-ctx.Done():
			svc.CancelJob(job.ID)
			fmt.Println("\ncancelled by user")
			return
		case <-ticker.C:
			cur, ok := svc.GetJob(job.ID)
			if !ok {
				fmt.Println("\njob no longer tracked")
				return
			}
			if cur.Progress != lastPct {
				lastPct = cur.Progress
				fmt.Printf("\rprogress=%4.1f%% phase=%-16s transferred=%s",
					cur.Progress, cur.Phase, humanBytes(cur.Transferred))
			}
			if isDone(cur.Status) {
				fmt.Printf("\nstatus=%s phase=%s error=%q\n", cur.Status, cur.Phase, cur.Error)
				if cur.Status == transfer.StatusCompleted {
					fmt.Printf("OK: %d bytes transferred (100%%)\n", cur.FileSize)
					return
				}
				os.Exit(1)
			}
		}
	}
}

func isDone(s transfer.JobStatus) bool {
	return s == transfer.StatusCompleted || s == transfer.StatusFailed || s == transfer.StatusCancelled
}

// humanBytes renders a byte count compactly (KB/MB/GB), independent of the
// package-internal formatter.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), units[exp])
}
