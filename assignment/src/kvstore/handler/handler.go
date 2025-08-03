package handler

import (
	"encoding/json"
	"fmt"
	"kvstore/hash"
	"kvstore/model"
	"kvstore/store"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	SelfURL     string
	HashRing    *hash.HashRing
	Store       store.KeyValueStore
	Replicas    int
	WriteQuorum int
	ReadQuorum  int
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
	isForwarded := r.Header.Get("X-From-Node") == "true"
	targets := h.getResponsibleNodes(key)

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		value := r.URL.Query().Get("value")
		h.handlePost(isForwarded, targets, key, value, r, w)
	case http.MethodGet:
		h.handleGet(isForwarded, targets, key, r, w)
	default:
		writeJSONError(w, http.StatusInternalServerError, "Method not supported")
		return
	}
}

func (h *Handler) handleGet(isForwarded bool, targets []string, key string, r *http.Request, w http.ResponseWriter) {
	var valueVersion model.ValueVersion
	if isForwarded {
		valueVersion, ok := h.Store.Get(key)
		if !ok {
			errorMsg := fmt.Sprintf("Key %v not found on node %v", key, h.SelfURL)
			writeJSONError(w, http.StatusInternalServerError, errorMsg)
			return
		}
		log.Printf("GET [%v -> %v] local", key, valueVersion)
	} else {
		latestValue := ""
		latestTimestamp := int64(0)

		var successNodes []string
		for _, target := range targets {
			var val string
			var ts int64
			if target == h.SelfURL {
				value, ok := h.Store.Get(key)
				if !ok {
					errorMsg := fmt.Sprintf("Key %v not found on node %v", key, h.SelfURL)
					writeJSONError(w, http.StatusInternalServerError, errorMsg)
					return
				}
				val = value.Value
				ts = value.Timestamp
				successNodes = append(successNodes, target)
			} else {
				value, err := h.forward(target, r)
				val = value.Value
				ts = value.Timestamp
				if err != nil {
					log.Printf("Error forwarding request to %v: %v", target, err)
					continue
				}
				successNodes = append(successNodes, target)
			}
			if ts > latestTimestamp {
				latestValue = val
				latestTimestamp = ts
			}
		}
		if len(successNodes) < h.ReadQuorum {
			errorMsg := fmt.Sprintf("Read quorum not met: %d < %d, success nodes: %v", len(successNodes), h.ReadQuorum, successNodes)
			writeJSONError(w, http.StatusInternalServerError, errorMsg)
			return
		}
		valueVersion = model.ValueVersion{
			Value:     latestValue,
			Timestamp: latestTimestamp,
		}
		log.Printf("GET [%v -> %v] from %v nodes: %v", key, valueVersion, len(successNodes), successNodes)
	}
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

func (h *Handler) handlePost(isForwarded bool, targets []string, key string, value string, r *http.Request, w http.ResponseWriter) {
	if value == "" {
		http.Error(w, "Missing value", http.StatusBadRequest)
		return
	}
	valueVersion := model.ValueVersion{
		Value:     value,
		Timestamp: time.Now().UnixNano(),
	}
	if isForwarded {
		h.Store.Put(key, valueVersion)
		log.Printf("PUT [%v -> %v] from forwarded request", key, value)
	} else {
		var successNodes []string
		for _, target := range targets {
			if target == h.SelfURL {
				h.Store.Put(key, valueVersion)
				successNodes = append(successNodes, target)
			} else {
				_, err := h.forward(target, r)
				if err != nil {
					log.Printf("Error forwarding request to %v: %v", target, err)
					continue
				}
				successNodes = append(successNodes, target)
			}
		}
		if len(successNodes) < h.WriteQuorum {
			errorMsg := fmt.Sprintf("Write quorum not met: %d < %d, success nodes: %v", len(successNodes), h.WriteQuorum, successNodes)
			writeJSONError(w, http.StatusInternalServerError, errorMsg)
			return
		}
		log.Printf("PUT [%v -> %v] to %d nodes: %v", key, value, len(successNodes), successNodes)
	}
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

func (h *Handler) forward(target string, r *http.Request) (model.ValueVersion, error) {
	client := &http.Client{}
	targetURL := target + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequest(r.Method, targetURL, nil)
	if err != nil {
		return model.ValueVersion{}, fmt.Errorf("create request failed: %v", err)
	}
	req.Header = r.Header
	req.Header.Set("X-From-Node", "true")
	resp, err := client.Do(req)
	if err != nil {
		return model.ValueVersion{}, fmt.Errorf("do request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return model.ValueVersion{}, fmt.Errorf("request failed: %v", resp.Status)
	}
	var valueVersion model.ValueVersion
	if err := json.NewDecoder(resp.Body).Decode(&valueVersion); err != nil {
		return model.ValueVersion{}, fmt.Errorf("decode response failed: %v", err)
	}
	return valueVersion, nil
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

func (h *Handler) getResponsibleNodes(key string) []string {
	return h.HashRing.GetNodesForKey(key, h.Replicas)
}
