package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

func main() {
	// Define the subdomain you want this client to represent.
	subdomain := "localhost:8080"

	// Construct the WebSocket URL to connect to the SkyConnect server.
	// Since everything is local, we use "localhost:8080".
	serverURL := fmt.Sprintf("ws://localhost:8080/connect?subdomain=%s", url.QueryEscape(subdomain))

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("Error connecting to SkyConnect server:", err)
	}
	defer conn.Close()

	// Continuously listen for forwarded requests from the SkyConnect server.
	for {
		// Read a message from the server (which contains the forwarded HTTP request path).
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading from server:", err)
			break
		}
		requestPath := string(message)
		log.Printf("Received forwarded request: %s\n", requestPath)

		// Forward the request to your local server (running on localhost:3000).
		localURL := fmt.Sprintf("http://localhost:3000%s", requestPath)
		resp, err := http.Get(localURL)
		if err != nil {
			log.Println("Error contacting local server:", err)
			conn.WriteMessage(websocket.TextMessage, []byte("Error contacting local server"))
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Println("Error reading response from local server:", err)
			conn.WriteMessage(websocket.TextMessage, []byte("Error reading local response"))
			continue
		}

		// Send the local server's response back to the SkyConnect server.
		err = conn.WriteMessage(websocket.TextMessage, body)
		if err != nil {
			log.Println("Error sending response to server:", err)
			break
		}
	}
}
