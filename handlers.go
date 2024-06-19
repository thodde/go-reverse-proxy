package main

import (
	"io"
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
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all connections
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

// This took me some googling to get working gracefully. Also the gorilla websocket repo has some awesome examples.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Get the next backend server
	target := getNextBackend()
	// Convert the target URL to a WebSocket URL
	targetURL := "ws" + strings.TrimPrefix(target, "http")
	
	log.Printf("Proxying request to: %s\n", targetURL)

	// Upgrade the HTTP connection to a WebSocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
	}
	defer ws.Close()

	log.Printf("WebSocket connection established\n")

	activeWebsockets.Add(1)
	defer activeWebsockets.Done()

	backendURL := targetURL + "/ws"
	// Setup headers for WebSocket handshake
	requestHeader := http.Header{
		"Origin": {r.Header.Get("Origin")},
	}

	// Establish connection to backend WebSocket
	backendConn, resp, err := websocket.DefaultDialer.Dial(backendURL, requestHeader)
	if err != nil {
		log.Printf("WebSocket Dial error: %v", err)
		if resp != nil {
			// Write the response to the client
			http.Error(w, resp.Status, resp.StatusCode)
		} else {
			http.Error(w, "Could not connect to backend WebSocket", http.StatusInternalServerError)
		}
		return
	}
	defer backendConn.Close()

	log.Printf("WebSocket connection established to backend: %s\n", targetURL)

	errChan := make(chan error, 2)

	// Proxy WebSocket traffic in both directions
	go reader(ws, backendConn, errChan)
	go reader(backendConn, ws, errChan)

	// Wait for any error from either direction
	err = <-errChan
	if err != nil && err != io.EOF {
		log.Printf("WebSocket proxy error: %v\n", err)
	}

	log.Printf("Closing WebSocket connections\n")
}

func reader(src, dst *websocket.Conn, errChan chan error) {
	for {
		messageType, msg, err := src.ReadMessage()
		if err != nil {
			// If the error is not a normal close, send it to the error channel
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
				errChan <- err
			}
			log.Println("Client closed connection")
			return
		}

		log.Printf("Received message from %s: %s\n", src.RemoteAddr(), msg)

		err = dst.WriteMessage(messageType, msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
				errChan <- err
			}
			log.Println("Client closed connection")
			return
		}
	}
}
