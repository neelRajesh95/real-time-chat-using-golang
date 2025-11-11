package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (in bytes)
	maxMessageSize = 5120
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for POC (in production, validate origin)
		return true
	},
}

// Client represents a connected WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe access
	mu sync.RWMutex
}

// Message represents a chat message
type Message struct {
	Type        string `json:"type"`
	UserID      string `json:"userID,omitempty"`
	Username    string `json:"username,omitempty"`
	Content     string `json:"content,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
	ClientCount int    `json:"clientCount,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Filesize    int64  `json:"filesize,omitempty"`
	Filetype    string `json:"filetype,omitempty"`
	Filedata    string `json:"filedata,omitempty"`
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

			// Send client count to all clients
			h.broadcastClientCount()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

			// Send client count to all clients
			h.broadcastClientCount()

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			clientCount := len(clients)
			h.mu.RUnlock()

			log.Printf("Hub: Broadcasting message to %d clients, message length: %d", clientCount, len(message))
			// Broadcast to all clients (including sender)
			sentCount := 0
			for i, client := range clients {
				select {
				case client.send <- message:
					sentCount++
					log.Printf("Hub: Message queued to client %d (userID=%s) send channel", i, client.userID)
				default:
					// Client's send buffer is full, close the connection
					log.Printf("Client %s send buffer full, closing connection", client.userID)
					h.mu.Lock()
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						close(client.send)
					}
					h.mu.Unlock()
				}
			}
			log.Printf("Hub: Message queued to %d/%d clients' send channels", sentCount, clientCount)
		}
	}
}

// broadcastClientCount sends the current client count to all connected clients (non-blocking)
func (h *Hub) broadcastClientCount() {
	h.mu.RLock()
	count := len(h.clients)
	h.mu.RUnlock()

	// Only broadcast if there are clients connected
	if count == 0 {
		return
	}

	message := Message{
		Type:        "client_count",
		ClientCount: count,
		Timestamp:   time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling client count: %v", err)
		return
	}

	// Send non-blocking to avoid deadlocks
	select {
	case h.broadcast <- data:
		log.Printf("Client count broadcast sent successfully")
	default:
		log.Printf("Broadcast channel full, skipping client count update")
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		log.Printf("ReadPump exiting for client %s", c.userID)
		c.hub.unregister <- c
		c.conn.Close()
	}()

	log.Printf("ReadPump started for client %s", c.userID)
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", c.userID, err)
			} else {
				log.Printf("ReadPump error for client %s (normal close): %v", c.userID, err)
			}
			break
		}

		log.Printf("ReadPump: Received message type=%d, length=%d bytes from client %s", messageType, len(messageBytes), c.userID)
		log.Printf("ReadPump: Raw message data: %s", string(messageBytes))

		// Parse incoming message
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v, raw: %s", err, string(messageBytes))
			continue
		}

		// Ensure userID is set to the client's userID (security: prevent spoofing)
		msg.UserID = c.userID

		// Handle timestamp: convert milliseconds to seconds if needed
		if msg.Timestamp == 0 {
			msg.Timestamp = time.Now().Unix()
		} else if msg.Timestamp > 9999999999 {
			// Timestamp is in milliseconds, convert to seconds
			msg.Timestamp = msg.Timestamp / 1000
		}

		// Set message type if not set
		if msg.Type == "" {
			msg.Type = "message"
		}

		// Validate message content
		if msg.Content == "" && msg.Type == "message" {
			log.Printf("Received empty message from %s, ignoring", msg.Username)
			continue
		}

		// Skip validation for typing and file messages
		if msg.Type == "file" && msg.Filename == "" {
			log.Printf("Received file message without filename from %s, ignoring", msg.Username)
			continue
		}

		// Log received message for debugging
		log.Printf("Received %s message from userID=%s username=%s content='%s'", 
			msg.Type, c.userID, msg.Username, msg.Content)

		// Broadcast message to all clients (including sender)
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			continue
		}

		// Get client count before broadcasting
		c.hub.mu.RLock()
		clientCount := len(c.hub.clients)
		c.hub.mu.RUnlock()
		
		log.Printf("Queuing message to broadcast channel for %d clients", clientCount)
		log.Printf("Message data to broadcast: %s", string(data))
		c.hub.broadcast <- data
		log.Printf("Message queued successfully to broadcast channel")
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send message as a single WebSocket text frame
			log.Printf("WritePump: Sending message to client %s, message length: %d", c.userID, len(message))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error to client %s: %v", c.userID, err)
				return
			}
			log.Printf("WritePump: Message sent successfully to client %s", c.userID)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error to client %s: %v", c.userID, err)
				return
			}
		}
	}
}

// serveWS handles WebSocket requests from clients
func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Printf("New WebSocket connection from %s", r.RemoteAddr)

	// Get user ID from query parameter or generate one
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		userID = generateUserID()
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	log.Printf("Registering client %s with hub", userID)
	client.hub.register <- client
	log.Printf("Client %s registered, starting ReadPump and WritePump", userID)

	// Start goroutines for reading and writing
	// IMPORTANT: ReadPump must handle incoming messages, WritePump handles outgoing
	go client.WritePump()
	go client.ReadPump()
	
	log.Printf("Client %s goroutines started", userID)
}

// generateUserID generates a simple user ID (in production, use a proper ID generator)
func generateUserID() string {
	return "user_" + time.Now().Format("20060102150405")
}

// handleHealth returns a simple health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"service": "chat-backend",
	})
}

// handleStats returns connection statistics
func handleStats(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hub.mu.RLock()
		clientCount := len(hub.clients)
		hub.mu.RUnlock()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"clients": clientCount,
			"version": "1.1.0",
			"timestamp": time.Now().Unix(),
		})
	}
}

func main() {
	hub := NewHub()
	go hub.Run()

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})

	// Health check endpoint
	http.HandleFunc("/health", handleHealth)
	
	// Stats endpoint
	http.HandleFunc("/stats", handleStats(hub))

	// Serve client.html at /client.html
	http.HandleFunc("/client.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "client.html")
	})

	// Serve client.html at root
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only serve client.html at root, return 404 for other paths
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "client.html")
		} else {
			http.NotFound(w, r)
		}
	})

	port := ":8080"
	log.Printf("========================================")
	log.Printf("Chat server starting on port %s", port)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", port)
	log.Printf("Health check: http://localhost%s/health", port)
	log.Printf("Stats: http://localhost%s/stats", port)
	log.Printf("Chat client: http://localhost%s/", port)
	log.Printf("========================================")
	log.Printf("Server is ready! Open browser to test.")
	log.Printf("========================================")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}

