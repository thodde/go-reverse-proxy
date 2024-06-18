package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/websocket"
)

var (
	backends       = []string{}
	currentBackend uint32
	validTokens    = map[string]bool{
		"valid-token-1": true,
		"valid-token-2": true,
	}
)

func main() {
	configFile := "config.json"
	// get the backend config from the config file
	config := readConfig(configFile)

	// fill the list of backends from the config
	for _, config := range config.Backends {
		backends = append(backends, config.Address)
	}

	mux := http.NewServeMux()
	// Add two endpoints to the mux
	mux.HandleFunc("/", authMiddleware(reverseProxyHandler))
	mux.Handle("/ws", websocket.Handler(wsHandler))

	// Create a new HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	log.Println("Server started")

	// Wait for the stop signal
	<-stop
	log.Println("Server stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	log.Println("Server stopped")
}

// authMiddleware is a middleware that checks if the request has a valid token
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Auth-Token")
		if !validTokens[token] {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}
