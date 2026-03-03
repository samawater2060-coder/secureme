package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// Store defines the persistence interface.
type Store interface {
	UpsertDevice(ctx context.Context, d *Device) error
	UpdateLastSeen(ctx context.Context, id string) error
	ListDevices(ctx context.Context) ([]*Device, error)
	GetDevice(ctx context.Context, id string) (*Device, error)
}

// MemoryStore is a thread-safe in-memory Store implementation.
type MemoryStore struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

// NewMemoryStore returns an initialised MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{devices: make(map[string]*Device)}
}

func (m *MemoryStore) UpsertDevice(_ context.Context, d *Device) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[d.ID] = d
	return nil
}

func (m *MemoryStore) UpdateLastSeen(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	dev, ok := m.devices[id]
	if !ok {
		return ErrNotFound
	}
	dev.LastSeen = time.Now().UTC()
	return nil
}

func (m *MemoryStore) ListDevices(_ context.Context) ([]*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Device, 0, len(m.devices))
	for _, d := range m.devices {
		result = append(result, d)
	}
	return result, nil
}

func (m *MemoryStore) GetDevice(_ context.Context, id string) (*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.devices[id]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}
