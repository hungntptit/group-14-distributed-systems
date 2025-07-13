package store

import (
	"testing"
)

func TestMemoryStore_PutAndGet(t *testing.T) {
	store := NewMemoryStore()

	store.Put("foo", "bar")
	value, ok := store.Get("foo")

	if !ok {
		t.Errorf("Expected key 'foo' to exist")
	}
	if value != "bar" {
		t.Errorf("Expected value 'bar', got '%s'", value)
	}
}

func TestMemoryStore_GetMissingKey(t *testing.T) {
	store := NewMemoryStore()

	_, ok := store.Get("missing")
	if ok {
		t.Errorf("Expected key 'missing' to not exist")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()

	store.Put("temp", "value")
	store.Delete("temp")

	_, ok := store.Get("temp")
	if ok {
		t.Errorf("Expected key 'temp' to be deleted")
	}
}
