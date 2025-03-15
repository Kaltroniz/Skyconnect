package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// clients maps subdomains (strings) to WebSocket connections.
var (
	clients      = make(map[string]*websocket.Conn)
	clientsMutex = &sync.Mutex{}
)

// upgrader is used to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	// Allow all origins for simplicity.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleClient upgrades a connection, registers the client with its subdomain, and listens for messages.
func handleClient(w http.ResponseWriter, r *http.Request) {
	// Upgrade the connection to a WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	// No echo loop here—just register the connection.
	subdomain := r.URL.Query().Get("subdomain")
	if subdomain == "" {
		subdomain = fmt.Sprintf("client%d", len(clients)+1)
	}

	clientsMutex.Lock()
	clients[subdomain] = conn
	clientsMutex.Unlock()

	log.Printf("Client connected with subdomain: %s\n", subdomain)

	// Keep the connection open without echoing messages.
	// A simple blocking read to keep the connection alive:
	select {}
}

// handleProxy is the HTTP handler for incoming external requests.
// It uses the Host header (subdomain) to determine which client connection should receive the request.
func handleProxy(w http.ResponseWriter, r *http.Request) {
	// Assume the Host header contains the subdomain (e.g. your-app.skyconnect.com).
	subdomain := r.Host

	clientsMutex.Lock()
	clientConn, ok := clients[subdomain]
	clientsMutex.Unlock()

	if !ok {
		http.Error(w, "Subdomain not found", http.StatusNotFound)
		return
	}

	// Forward the incoming HTTP request path to the client via WebSocket.
	err := clientConn.WriteMessage(websocket.TextMessage, []byte(r.URL.String()))
	if err != nil {
		log.Println("Error forwarding request to client:", err)
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}

	// Wait for the client’s response (which is expected to come over the same WebSocket).
	_, response, err := clientConn.ReadMessage()
	if err != nil {
		log.Println("Error reading response from client:", err)
		http.Error(w, "Error reading response", http.StatusInternalServerError)
		return
	}

	// Write the response back to the external requester.
	w.Write(response)
}

func main() {
	// /connect endpoint for clients to establish WebSocket connections.
	http.HandleFunc("/connect", handleClient)
	// All other requests are treated as proxy requests.
	http.HandleFunc("/", handleProxy)

	log.Println("SkyConnect server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
