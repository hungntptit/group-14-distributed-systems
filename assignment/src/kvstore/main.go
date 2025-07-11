package main

import (
	"fmt"
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
}
