package store

import "sync"

type MemoryStore struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]string),
	}
}

func (m *MemoryStore) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func (m *MemoryStore) Put(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *MemoryStore) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}
