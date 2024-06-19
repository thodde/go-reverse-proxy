package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type BackendConfig struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Config struct {
	Backends []BackendConfig `json:"backends"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all connections
}

func main() {
	// read backends from the config file
	configFile := "config.json"
	config := readConfig(configFile)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	for _, backend := range config.Backends {
		wg.Add(1)
		go startServer(ctx, &wg, backend.Address, backend.Name)
	}

	// Wait for interrupt signal to gracefully shutdown the server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down servers...")

	cancel()
	wg.Wait()
	log.Println("Servers gracefully stopped.")
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

func startServer(ctx context.Context, wg *sync.WaitGroup, addr, name string) {
	defer wg.Done()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s received a request\n", name)
		fmt.Fprintf(w, "Hello from %s!", name)
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, addr)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// similar approach to graceful shutdown in the reverse proxy
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("%s: %v", name, err)
		}
	}()

	log.Printf("%s started on %s\n", name, addr)

	<-ctx.Done()
	log.Printf("%s shutting down...", name)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down server %s: %v", name, err)
	} else {
		log.Printf("Server %s stopped gracefully.", name)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, addr string) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	log.Printf("WebSocket connection established on %s", addr)

	for {
		// Read message from client
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			log.Println("Client closed connection")
			break
		}
		log.Printf("Received message: %s", message)
		err = ws.WriteMessage(messageType, message)
		if err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}
	log.Printf("WebSocket connection closed on %s", addr)
}
