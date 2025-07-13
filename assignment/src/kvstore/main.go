package main

import (
	"fmt"
	"kvstore/hash"
	"kvstore/network"
	"kvstore/store"
	"log"
	"net/http"
	"os"
	"strings"
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

func main() {
	config := loadConfig()

	fmt.Println("SELF_URL:", config.SelfURL)
	fmt.Println("PORT    :", config.Port)
	fmt.Println("PEERS   :", config.Peers)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			return
		}
	})

	r := hash.NewHashRing(config.Peers, 100)
	r.AddNode(config.SelfURL)

	kvStore := store.NewMemoryStore()

	h := &network.Handler{
		SelfURL:  config.SelfURL,
		HashRing: r,
		Store:    kvStore,
	}

	http.Handle("/kv", h)

	addr := ":" + config.Port
	log.Printf("Listening on %s...", config.Port)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
