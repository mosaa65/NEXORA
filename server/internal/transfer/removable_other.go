//go:build !windows

package transfer

import (
	"context"
)

func (s *Service) discoverRemovableDrives(ctx context.Context) ([]Device, error) {
	return []Device{}, nil
}
