package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"golang.org/x/net/websocket"
)

func reverseProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Get the next backend server
	target := getNextBackend()
	targetURL, _ := url.Parse(target)
	// Create a new reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

func wsHandler(ws *websocket.Conn) {
	// Get the next backend server
	target := getNextBackend()
	// Convert the target URL to a WebSocket URL
	targetURL := "ws" + strings.TrimPrefix(target, "http")
	// Dial the backend WebSocket server
	backendWS, err := websocket.Dial(targetURL+"/ws", "", "http://localhost:8080")
	if err != nil {
		log.Println("WebSocket Dial error:", err)
		return
	}
	defer backendWS.Close()

	go func() {
		io.Copy(backendWS, ws)
		ws.Close()
	}()
	io.Copy(ws, backendWS)
}
