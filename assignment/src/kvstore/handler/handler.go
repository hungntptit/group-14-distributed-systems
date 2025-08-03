package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"kvstore/hash"
	"kvstore/logging"
	"kvstore/model"
	"kvstore/store"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const GossipInterval = 3 * time.Second
const PeerTimeout = 15 * time.Second

type Handler struct {
	SelfURL     string
	HashRing    *hash.HashRing
	Store       store.KeyValueStore
	Replicas    int
	WriteQuorum int
	ReadQuorum  int

	Peers map[string]*model.PeerInfo
	Mu    sync.Mutex
}

type KVResponse struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type GossipMessage struct {
	Sender string                     `json:"sender"`
	Peers  map[string]*model.PeerInfo `json:"peers"`
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
	if err != nil {
		logging.Errorf("Error encoding error response: %v", err)
		return
	}
}

func (h *Handler) getResponsibleNodes(key string) []string {
	return h.HashRing.GetNodesForKey(key, h.Replicas)
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
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not supported")
		return
	}
}

func (h *Handler) handleGet(isForwarded bool, targets []string, key string, r *http.Request, w http.ResponseWriter) {
	var valueVersion model.ValueVersion
	if isForwarded {
		valueVersion, ok := h.Store.Get(key)
		if !ok {
			errorMsg := fmt.Sprintf("Key %v not found on node %v", key, h.SelfURL)
			writeJSONError(w, http.StatusNotFound, errorMsg)
			return
		}
		logging.Infof("GET [%v -> %v] local", key, valueVersion)
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
					writeJSONError(w, http.StatusNotFound, errorMsg)
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
					logging.Errorf("Error forwarding request to %v: %v", target, err)
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
		logging.Infof("GET [%v -> %v] from %v nodes: %v", key, valueVersion, len(successNodes), successNodes)
	}
	resp := KVResponse{
		Key:       key,
		Value:     valueVersion.Value,
		Timestamp: valueVersion.Timestamp,
	}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		logging.Errorf("Error encoding response: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Error encoding response")
		return
	}

}

func (h *Handler) handlePost(isForwarded bool, targets []string, key string, value string, r *http.Request, w http.ResponseWriter) {
	if value == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing value")
		return
	}
	valueVersion := model.ValueVersion{
		Value:     value,
		Timestamp: time.Now().UnixNano(),
	}
	if isForwarded {
		h.Store.Put(key, valueVersion)
		logging.Infof("PUT [%v -> %v] from forwarded request", key, value)
	} else {
		var successNodes []string
		for _, target := range targets {
			if target == h.SelfURL {
				h.Store.Put(key, valueVersion)
				successNodes = append(successNodes, target)
			} else {
				_, err := h.forward(target, r)
				if err != nil {
					logging.Errorf("Error forwarding request to %v: %v", target, err)
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
		logging.Infof("PUT [%v -> %v] to %d nodes: %v", key, value, len(successNodes), successNodes)
	}
	resp := KVResponse{
		Key:       key,
		Value:     valueVersion.Value,
		Timestamp: valueVersion.Timestamp,
	}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		logging.Errorf("Error encoding response: %v", err)
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
			logging.Errorf("Error encoding response: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Error encoding response")
			return
		}
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not supported")
		return
	}
}

func (h *Handler) GossipHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var msg GossipMessage
		err := json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Error decoding request")
			return
		}
		logging.Debugf("Gossip received from %v", msg.Sender)

		for url, incomingPeer := range msg.Peers {
			local, exists := h.Peers[url]
			if !exists || incomingPeer.LastSeen.After(local.LastSeen) {
				h.Peers[url] = &model.PeerInfo{
					URL:      url,
					LastSeen: incomingPeer.LastSeen,
				}
			}
		}
		h.updateHashRingPeers()
		h.Peers[msg.Sender].LastSeen = time.Now()

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not supported")
		return
	}
}

func (h *Handler) SendGossip(target string) {
	msg := GossipMessage{
		Sender: h.SelfURL,
		Peers:  h.Peers,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		logging.Errorf("Error encoding gossip message: %v", err)
		return
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(target+"/kv/gossip", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logging.Errorf("Error sending gossip to %s: %v", target, err)
		return
	}
	defer resp.Body.Close()
	logging.Debugf("Sent gossip to %s, status: %s", target, resp.Status)
	h.Peers[target].LastSeen = time.Now()
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		return
	}
}

func (h *Handler) updateHashRingPeers() {
	changed := false
	for _, peer := range h.Peers {
		if time.Since(peer.LastSeen) >= PeerTimeout {
			h.HashRing.RemoveNode(peer.URL)
			changed = true
			continue
		}
		if !h.HashRing.ContainsPeer(peer.URL) {
			h.HashRing.AddNode(peer.URL)
			changed = true
		}
	}
	if changed {
		logging.Debugf("Gossip updated peers: %v", h.Peers)
	}
}

func (h *Handler) StartGossiping() {
	go func() {
		ticker := time.NewTicker(GossipInterval)
		defer ticker.Stop()

		for range ticker.C {
			if peerURL, ok := h.PickRandomPeerToGossip(); ok {
				h.SendGossip(peerURL)
			}
		}
	}()
}

func (h *Handler) PickRandomPeerToGossip() (string, bool) {
	var candidates []string
	for url, peer := range h.Peers {
		if url == h.SelfURL {
			continue
		}
		if time.Since(peer.LastSeen) <= PeerTimeout {
			candidates = append(candidates, url)
		}
	}
	if len(candidates) == 0 {
		return "", false
	}
	randomIndex := rand.Intn(len(candidates))
	return candidates[randomIndex], true
}
