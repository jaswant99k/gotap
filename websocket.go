// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrWebSocketUpgradeFailed is returned when WebSocket upgrade fails
	ErrWebSocketUpgradeFailed = errors.New("websocket upgrade failed")
	// ErrConnectionClosed is returned when connection is closed
	ErrConnectionClosed = errors.New("connection closed")
)

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	// ReadBufferSize specifies the size of the read buffer
	ReadBufferSize int

	// WriteBufferSize specifies the size of the write buffer
	WriteBufferSize int

	// HandshakeTimeout specifies the duration for the handshake to complete
	HandshakeTimeout time.Duration

	// CheckOrigin returns true if the request Origin header is acceptable
	CheckOrigin func(r *http.Request) bool

	// Error defines a function to handle errors
	Error func(c *Context, status int, err error)

	// Subprotocols specifies the server's supported protocols
	Subprotocols []string
}

// WebSocketHandler defines the function signature for WebSocket handlers
type WebSocketHandler func(*WebSocketConn)

// WebSocketConn wraps gorilla/websocket connection with additional features
type WebSocketConn struct {
	*websocket.Conn
	mu       sync.Mutex
	writeMu  sync.Mutex
	Context  *Context
	closed   bool
	sendChan chan []byte
}

// WSUpgrader is the default WebSocket upgrader
var WSUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins by default
	},
}

// WebSocket upgrades the HTTP connection to WebSocket and handles it
func (c *Context) WebSocket(handler WebSocketHandler) {
	c.WebSocketWithConfig(WebSocketConfig{}, handler)
}

// WebSocketWithConfig upgrades with custom configuration
func (c *Context) WebSocketWithConfig(config WebSocketConfig, handler WebSocketHandler) {
	// Set defaults
	if config.ReadBufferSize == 0 {
		config.ReadBufferSize = 1024
	}
	if config.WriteBufferSize == 0 {
		config.WriteBufferSize = 1024
	}
	if config.HandshakeTimeout == 0 {
		config.HandshakeTimeout = 10 * time.Second
	}
	if config.CheckOrigin == nil {
		config.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}
	if config.Error == nil {
		config.Error = func(c *Context, status int, err error) {
			c.String(status, err.Error())
		}
	}

	// Create upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:   config.ReadBufferSize,
		WriteBufferSize:  config.WriteBufferSize,
		HandshakeTimeout: config.HandshakeTimeout,
		CheckOrigin:      config.CheckOrigin,
		Subprotocols:     config.Subprotocols,
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		config.Error(c, http.StatusBadRequest, err)
		return
	}

	// Create WebSocket connection wrapper
	wsConn := &WebSocketConn{
		Conn:     conn,
		Context:  c,
		sendChan: make(chan []byte, 256),
	}

	// Start write pump
	go wsConn.writePump()

	// Handle WebSocket
	handler(wsConn)

	// Cleanup
	wsConn.Close()
}

// SendText sends a text message
func (ws *WebSocketConn) SendText(message string) error {
	return ws.Send([]byte(message))
}

// SendJSON sends a JSON message
func (ws *WebSocketConn) SendJSON(v interface{}) error {
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()

	if ws.closed {
		return ErrConnectionClosed
	}

	return ws.WriteJSON(v)
}

// Send sends a binary message
func (ws *WebSocketConn) Send(message []byte) error {
	if ws.closed {
		return ErrConnectionClosed
	}

	select {
	case ws.sendChan <- message:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

// ReadText reads a text message
func (ws *WebSocketConn) ReadText() (string, error) {
	messageType, message, err := ws.ReadMessage()
	if err != nil {
		return "", err
	}

	if messageType != websocket.TextMessage {
		return "", errors.New("not a text message")
	}

	return string(message), nil
}

// ReadJSON reads a JSON message
func (ws *WebSocketConn) ReadJSON(v interface{}) error {
	return ws.Conn.ReadJSON(v)
}

// Close closes the WebSocket connection
func (ws *WebSocketConn) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.closed {
		return nil
	}

	ws.closed = true
	close(ws.sendChan)

	// Send close message
	ws.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))

	return ws.Conn.Close()
}

// IsClosed returns true if connection is closed
func (ws *WebSocketConn) IsClosed() bool {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.closed
}

// writePump handles outgoing messages
func (ws *WebSocketConn) writePump() {
	for message := range ws.sendChan {
		ws.writeMu.Lock()
		if err := ws.WriteMessage(websocket.TextMessage, message); err != nil {
			ws.writeMu.Unlock()
			return
		}
		ws.writeMu.Unlock()
	}
}

// SetReadDeadline sets the read deadline
func (ws *WebSocketConn) SetReadDeadline(t time.Time) error {
	return ws.Conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline
func (ws *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return ws.Conn.SetWriteDeadline(t)
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketConn]bool
	broadcast  chan []byte
	register   chan *WebSocketConn
	unregister chan *WebSocketConn
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	hub := &WebSocketHub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketConn),
		unregister: make(chan *WebSocketConn),
		clients:    make(map[*WebSocketConn]bool),
	}

	go hub.run()

	return hub
}

// run handles hub operations
func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if !client.IsClosed() {
					client.Send(message)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register registers a new client
func (h *WebSocketHub) Register(client *WebSocketConn) {
	h.register <- client
}

// Unregister unregisters a client
func (h *WebSocketHub) Unregister(client *WebSocketConn) {
	h.unregister <- client
}

// Broadcast sends a message to all clients
func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastJSON sends a JSON message to all clients
func (h *WebSocketHub) BroadcastJSON(v interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if !client.IsClosed() {
			client.SendJSON(v)
		}
	}
}

// ClientCount returns the number of connected clients
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Clients returns all connected clients
func (h *WebSocketHub) Clients() []*WebSocketConn {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*WebSocketConn, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}

	return clients
}

// Close closes all connections
func (h *WebSocketHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		client.Close()
		delete(h.clients, client)
	}
}
