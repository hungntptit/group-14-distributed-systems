package main

import (
	"fmt"
	"kvstore/handler"
	"kvstore/hash"
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

func SetupRoutes(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/kv", h)
	mux.HandleFunc("/kv/all", h.GetAllHandler)
	return mux
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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

	r := hash.NewHashRing(config.Peers, 1)
	r.AddNode(config.SelfURL)

	kvStore := store.NewMemoryStore()

	h := &handler.Handler{
		SelfURL:     config.SelfURL,
		HashRing:    r,
		Store:       kvStore,
		Replicas:    3,
		ReadQuorum:  2,
		WriteQuorum: 2,
	}

	router := SetupRoutes(h)

	addr := ":" + config.Port
	log.Printf("Listening on %s...", config.Port)

	err := http.ListenAndServe(addr, router)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
