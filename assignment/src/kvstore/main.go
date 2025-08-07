package main

import (
	"fmt"
	"kvstore/handler"
	"kvstore/hash"
	"kvstore/logging"
	"kvstore/model"
	"kvstore/store"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	SelfURL string
	Port    string
	Peers   []string
}

func loadConfig() Config {
	selfURL := os.Getenv("SELF_URL")
	port := os.Getenv("PORT")
	peersRaw := os.Getenv("PEERS")

	if selfURL == "" || port == "" || peersRaw == "" {
		fmt.Println("Missing environment variables. Required: SELF_URL, PORT, PEERS")
		os.Exit(1)
	}

	peers := strings.Split(peersRaw, ",")

	return Config{
		SelfURL: selfURL,
		Port:    port,
		Peers:   peers,
	}
}

func SetupRoutes(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.HealthHandler)
	mux.Handle("/kv", h)
	mux.HandleFunc("/kv/all", h.GetAllHandler)
	mux.HandleFunc("/kv/gossip", h.GossipHandler)
	return mux
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logging.InitLogger(false)

	config := loadConfig()

	logging.Infof("SELF_URL: %s", config.SelfURL)
	logging.Infof("PORT    : %s", config.Port)
	logging.Infof("PEERS   : %s", config.Peers)

	hr := hash.NewHashRing(config.Peers, 1)
	hr.AddNode(config.SelfURL)

	kvStore := store.NewMemoryStore()

	peers := make(map[string]*model.PeerInfo)
	peers[config.SelfURL] = &model.PeerInfo{
		URL:      config.SelfURL,
		LastSeen: time.Now(),
	}
	for _, peer := range config.Peers {
		peers[peer] = &model.PeerInfo{
			URL:      peer,
			LastSeen: time.Now(),
		}
	}

	h := &handler.Handler{
		SelfURL:     config.SelfURL,
		HashRing:    hr,
		Store:       kvStore,
		Replicas:    3,
		ReadQuorum:  2,
		WriteQuorum: 2,
		Peers:       peers,
	}

	router := SetupRoutes(h)

	h.StartGossiping()

	addr := ":" + config.Port
	logging.Infof("Listening on %s...", config.Port)

	err := http.ListenAndServe(addr, router)
	if err != nil {
		logging.Errorf("Server failed: %v", err)
	}
}
