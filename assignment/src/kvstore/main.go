package main

import (
	"fmt"
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

	http.HandleFunc("/kv", kvHandler)

	addr := ":" + config.Port
	log.Printf("Listening on %s...", config.Port)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func kvHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "NOT IMPLEMENTED")
	if err != nil {
		return
	}
}
