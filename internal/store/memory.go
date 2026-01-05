package store

import (
	"sync"

	"github.com/mezotov/netdiscover/internal/model"
)

type MemoryStore struct {
	mu      sync.Mutex
	devices map[string]*model.Device
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		devices: make(map[string]*model.Device),
	}
}

func (m *MemoryStore) Get(id string) (*model.Device, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	return d, ok
}

func (m *MemoryStore) Save(d *model.Device) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[d.ID] = d
}

func (m *MemoryStore) All() []*model.Device {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*model.Device, 0, len(m.devices))
	for _, d := range m.devices {
		out = append(out, d)
	}
	return out
}
