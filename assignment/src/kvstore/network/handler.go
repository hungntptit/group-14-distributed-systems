package network

import (
	"encoding/json"
	"fmt"
	"io"
	"kvstore/hash"
	"kvstore/model"
	"kvstore/store"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	SelfURL  string
	HashRing *hash.HashRing
	Store    store.KeyValueStore
}

type KVResponse struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
	if err != nil {
		log.Printf("Error encoding error response: %v", err)
		return
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSONError(w, http.StatusInternalServerError, "Missing key")
		return
	}

	target := h.HashRing.GetNodeForKey(key)
	if target == "" {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Node for key %q not found", key))
		return
	}
	if target != h.SelfURL {
		h.forward(target, r, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		value := r.URL.Query().Get("value")
		h.handlePost(key, value, w)
	case http.MethodGet:
		h.handleGet(key, w)
	default:
		writeJSONError(w, http.StatusInternalServerError, "Method not supported")
		return
	}
}

func (h *Handler) handleGet(key string, w http.ResponseWriter) {
	value, ok := h.Store.Get(key)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Key not found")
		return
	}
	log.Printf("GET %s -> %s", key, value)
	resp := KVResponse{
		Key:       key,
		Value:     value.Value,
		Timestamp: value.Timestamp,
	}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Error encoding response")
		return
	}

}

func (h *Handler) handlePost(key string, value string, w http.ResponseWriter) {
	if value == "" {
		http.Error(w, "Missing value", http.StatusBadRequest)
		return
	}
	valueVersion := model.ValueVersion{
		Value:     value,
		Timestamp: time.Now().UnixNano(),
	}
	h.Store.Put(key, valueVersion)
	log.Printf("PUT %s -> %s", key, value)
	resp := KVResponse{
		Key:       key,
		Value:     valueVersion.Value,
		Timestamp: valueVersion.Timestamp,
	}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Error encoding response")
		return
	}
}

func (h *Handler) forward(target string, r *http.Request, w http.ResponseWriter) {
	client := &http.Client{}
	targetURL := target + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Error forwarding request")
	}
	req.Header = r.Header
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Error forwarding request")
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Error closing response body")
		}
	}(resp.Body)

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response body: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Error copying response body")
		return
	}
	log.Printf("Forwarded %s -> %s", r.URL.Path, targetURL)
}

func (h *Handler) GetAllHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		all := h.Store.All()
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(all)
		if err != nil {
			log.Printf("Error encoding response: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Error encoding response")
			return
		}
	default:
		writeJSONError(w, http.StatusInternalServerError, "Method not supported")
		return
	}
}
