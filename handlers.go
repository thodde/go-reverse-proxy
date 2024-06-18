package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

func reverseProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Get the next backend server
	target := getNextBackend()
	targetURL, _ := url.Parse(target)
	// Create a new reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	log.Printf("Proxying request to: %s\n", target)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

// I am not super confident with my websocket handler. It doesn't exit very gracefully and I needed to
// do some reading online to get this working.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Get the next backend server
	target := getNextBackend()
	// Convert the target URL to a WebSocket URL
	targetURL := "ws" + strings.TrimPrefix(target, "http")
	
	log.Printf("Proxying request to: %s\n", targetURL)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
	}

	log.Printf("WebSocket connection established\n")

	// format a string with the server name to send back to the client
	hello_str := fmt.Sprintf("Hello from %s!", targetURL)
	err = ws.WriteMessage(websocket.TextMessage, []byte(hello_str))
	if err != nil {
		log.Println(err)
	}
	// listen for new messages coming in through on the WebSocket connection
	reader(ws)
}

func reader(conn *websocket.Conn) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			return
		}

		log.Println("Received message:", string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println("Error writing message:", err)
			return
		}
	}
}
