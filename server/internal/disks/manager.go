package disks

import (
	"context"
	"nexora/server/internal/db"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) ScanDisks(ctx context.Context) ([]db.StorageDisk, error) {
	return detectDisks(ctx)
}
