package jobs

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
)

// WSMessage represents a message sent over WebSocket
type WSMessage struct {
	Type string      `json:"type"` // "job_update", "job_complete", "job_failed"
	Data interface{} `json:"data"`
}

// WebSocketClient wraps a connection with its own mutex
type WebSocketClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// WebSocketHub manages all active WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan WSMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

// Global hub instance
var hub *WebSocketHub

// InitWebSocketHub initializes the WebSocket hub
func InitWebSocketHub() {
	hub = &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WSMessage, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	go hub.run()
}

// run handles the WebSocket hub event loop
func (h *WebSocketHub) run() {
	for {
		select {
		case conn := <-h.register:
			client := &WebSocketClient{conn: conn}
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered. Total clients: %d", len(h.clients))

		case conn := <-h.unregister:
			h.mu.Lock()
			for client := range h.clients {
				if client.conn == conn {
					delete(h.clients, client)
					client.conn.Close()
					break
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client unregistered. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*WebSocketClient, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()
			
			// Write to each client with proper synchronization
			for _, client := range clients {
				go func(c *WebSocketClient) {
					c.mu.Lock()
					err := c.conn.WriteJSON(message)
					c.mu.Unlock()
					if err != nil {
						log.Printf("WebSocket write error: %v", err)
						h.unregister <- c.conn
					}
				}(client)
			}
		}
	}
}

// BroadcastJobUpdate sends a job update to all connected clients
func BroadcastJobUpdate(job *Job) {
	if hub == nil {
		return
	}

	message := WSMessage{
		Type: "job_update",
		Data: job.ToMap(),
	}

	select {
	case hub.broadcast <- message:
	default:
		log.Println("WebSocket broadcast channel full, dropping message")
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *websocket.Conn) {
	// Register client
	hub.register <- c

	// Keep connection alive and handle client messages
	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle ping/pong
		if messageType == websocket.PingMessage {
			if err := c.WriteMessage(websocket.PongMessage, nil); err != nil {
				break
			}
			continue
		}

		// Handle client messages (if needed in future)
		log.Printf("Received WebSocket message: %s", message)

		// Echo back or handle custom logic
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			if msgType, ok := msg["type"].(string); ok {
				switch msgType {
				case "ping":
					response := WSMessage{
						Type: "pong",
						Data: map[string]string{"status": "ok"},
					}
					if err := c.WriteJSON(response); err != nil {
						break
					}
				case "subscribe":
					// Could implement job-specific subscriptions
					response := WSMessage{
						Type: "subscribed",
						Data: map[string]string{"status": "ok"},
					}
					if err := c.WriteJSON(response); err != nil {
						break
					}
				}
			}
		}
	}

	// Unregister client
	hub.unregister <- c
}
