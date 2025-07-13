package network

import (
	"kvstore/hash"
	"kvstore/store"
	"net/http"
)

type Handler struct {
	SelfURL  string
	HashRing *hash.HashRing
	Store    store.KeyValueStore
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}
