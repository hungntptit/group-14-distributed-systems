package store

type KeyValueStore interface {
	Get(key string) (value string, ok bool)
	Put(key string, value string)
	Delete(key string)
}
