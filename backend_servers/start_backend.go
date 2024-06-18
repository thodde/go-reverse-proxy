package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

type BackendConfig struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Config struct {
	Backends []BackendConfig `json:"backends"`
}

func main() {
	// read backends from the config file
	configFile := "config.json"
	config := readConfig(configFile)

	for _, backend := range config.Backends {
		go startServer(backend.Address, backend.Name)
	}

	select {} // Block forever to keep the servers running -- not as graceful as the reverse proxy impelmentation but it works
}

func readConfig(file string) Config {
	content, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	log.Printf("Config file contents: %s", content)

	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	return config
}

func startServer(addr, name string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s received a request\n", name)
		fmt.Fprintf(w, "Hello from %s!", name)
	})

	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		log.Printf("WebSocket connection established on %s", addr)
	}))

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("%s started on %s\n", name, addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("%s: %v", name, err)
	}
}
