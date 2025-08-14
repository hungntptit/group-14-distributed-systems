package store

import (
	"kvstore/model"
	"sync"
)

type MemoryStore struct {
	data map[string]model.ValueVersion
	mu   sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]model.ValueVersion),
	}
}

func (m *MemoryStore) All() map[string]model.ValueVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data
}

func (m *MemoryStore) Get(key string) (model.ValueVersion, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func (m *MemoryStore) Put(key string, value model.ValueVersion) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *MemoryStore) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}
