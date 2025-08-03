package store

import "kvstore/model"

type KeyValueStore interface {
	Get(key string) (value model.ValueVersion, ok bool)
	Put(key string, value model.ValueVersion)
	Delete(key string)
	All() interface{}
}
