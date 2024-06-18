package main

import (
	"encoding/json"
	"log"
	"os"
	"sync/atomic"
)

// BackendConfig represents the configuration of a backend server
// This allows us to parse the JSON configuration file and adds flexibility so that we can add more fields in the future
type BackendConfig struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Config struct {
	Backends []BackendConfig `json:"backends"`
}

// getNextBackend uses a round robin algorithm to select the next backend server
func getNextBackend() string {
	next := atomic.AddUint32(&currentBackend, 1)
	return backends[next%uint32(len(backends))]
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
