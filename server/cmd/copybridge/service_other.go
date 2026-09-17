//go:build !windows

package main

import (
	"errors"
)

func isWindowsService() bool {
	return false
}

func runWindowsService() error {
	return errors.New("windows service is only supported on windows")
}

func installService() error {
	return errors.New("service installation is only supported on windows")
}

func uninstallService() error {
	return errors.New("service uninstallation is only supported on windows")
}

func startService() error {
	return errors.New("service control is only supported on windows")
}

func stopService() error {
	return errors.New("service control is only supported on windows")
}
