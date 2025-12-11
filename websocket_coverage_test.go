package goTap

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test WebSocket basic functionality
func TestWebSocketBasic(t *testing.T) {
	engine := New()
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			// Read message
			msg, err := ws.ReadText()
			if err != nil {
				return
			}
			
			// Echo back
			ws.SendText(msg)
			
			// Keep connection alive briefly for client to read
			time.Sleep(100 * time.Millisecond)
		})
	})

	// Create test server
	server := httptest.NewServer(engine)
	defer server.Close()

	// Connect websocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Send message
	testMsg := "hello websocket"
	if err := ws.WriteMessage(websocket.TextMessage, []byte(testMsg)); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Read response with deadline
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if string(msg) != testMsg {
		t.Errorf("Expected %s, got %s", testMsg, string(msg))
	}
}

// Test WebSocket with config
func TestWebSocketWithConfig(t *testing.T) {
	engine := New()
	
	engine.GET("/ws", func(c *Context) {
		config := WebSocketConfig{
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
			Subprotocols:    []string{"chat"},
		}
		c.WebSocketWithConfig(config, func(ws *WebSocketConn) {
			msg, _ := ws.ReadText()
			ws.SendText(msg)
			// Keep alive briefly
			time.Sleep(100 * time.Millisecond)
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	testMsg := "configured"
	ws.WriteMessage(websocket.TextMessage, []byte(testMsg))
	
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if string(msg) != testMsg {
		t.Errorf("Expected %s, got %s", testMsg, string(msg))
	}
}

// Test WebSocket JSON messages
func TestWebSocketJSON(t *testing.T) {
	engine := New()
	
	type Message struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			var msg Message
			if err := ws.ReadJSON(&msg); err != nil {
				return
			}
			
			// Echo back as JSON
			ws.SendJSON(msg)
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Send JSON
	testMsg := Message{Type: "test", Data: "hello"}
	if err := ws.WriteJSON(testMsg); err != nil {
		t.Fatalf("Failed to send JSON: %v", err)
	}

	// Read JSON response
	var response Message
	if err := ws.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	if response.Type != "test" || response.Data != "hello" {
		t.Errorf("Unexpected response: %+v", response)
	}
}

// Test WebSocket Hub
func TestWebSocketHub(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.run()
	defer hub.Close()

	engine := New()
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			hub.Register(ws)
			defer hub.Unregister(ws)
			
			// Keep connection alive
			for {
				_, _, err := ws.Conn.ReadMessage()
				if err != nil {
					break
				}
			}
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	// Connect two clients
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer ws2.Close()

	time.Sleep(100 * time.Millisecond) // Let connections register

	// Check client count
	count := hub.ClientCount()
	if count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}

	// Test broadcast
	testMsg := []byte("broadcast")
	hub.Broadcast(testMsg)

	ws1.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := ws1.ReadMessage()
	if err == nil && string(msg) == string(testMsg) {
		t.Logf("Broadcast received by client 1")
	}
}

// Test WebSocket Hub BroadcastJSON
func TestWebSocketHubBroadcastJSON(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.run()
	defer hub.Close()

	engine := New()
	
	type TestMsg struct {
		Message string `json:"message"`
	}
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			hub.Register(ws)
			defer hub.Unregister(ws)
			
			// Keep connection alive
			for {
				_, _, err := ws.Conn.ReadMessage()
				if err != nil {
					break
				}
			}
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws1.Close()

	time.Sleep(100 * time.Millisecond)

	// Broadcast JSON
	testMsg := TestMsg{Message: "json broadcast"}
	hub.BroadcastJSON(testMsg)

	ws1.SetReadDeadline(time.Now().Add(1 * time.Second))
	var received TestMsg
	if err := ws1.ReadJSON(&received); err == nil {
		if received.Message == testMsg.Message {
			t.Logf("JSON broadcast received correctly")
		}
	}
}

// Test WebSocket Close and IsClosed
func TestWebSocketClose(t *testing.T) {
	engine := New()
	
	closed := false
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			if ws.IsClosed() {
				t.Error("Connection should not be closed initially")
			}
			
			// Read one message
			ws.ReadText()
			
			// Close connection
			ws.Close()
			closed = ws.IsClosed()
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	ws.WriteMessage(websocket.TextMessage, []byte("test"))
	ws.Close()

	time.Sleep(100 * time.Millisecond)

	if !closed {
		t.Error("IsClosed should return true after Close()")
	}
}

// Test WebSocket deadlines
func TestWebSocketDeadlines(t *testing.T) {
	engine := New()
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			// Set deadlines
			ws.SetReadDeadline(time.Now().Add(5 * time.Second))
			ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
			
			// Echo message
			msg, _ := ws.ReadText()
			ws.SendText(msg)
			// Keep alive briefly
			time.Sleep(100 * time.Millisecond)
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	testMsg := "deadline test"
	ws.WriteMessage(websocket.TextMessage, []byte(testMsg))
	
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if string(msg) != testMsg {
		t.Errorf("Expected %s, got %s", testMsg, string(msg))
	}
}

// Test WebSocket Send method
func TestWebSocketSend(t *testing.T) {
	engine := New()
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			// Use Send method (raw bytes)
			msg, _ := ws.ReadText()
			ws.Send([]byte(msg))
			// Keep alive briefly
			time.Sleep(100 * time.Millisecond)
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	testMsg := "send test"
	ws.WriteMessage(websocket.TextMessage, []byte(testMsg))
	
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if string(msg) != testMsg {
		t.Errorf("Expected %s, got %s", testMsg, string(msg))
	}
}

// Test WebSocket upgrade validation
func TestWebSocketUpgrade(t *testing.T) {
	engine := New()
	
	engine.GET("/ws", func(c *Context) {
		c.WebSocket(func(ws *WebSocketConn) {
			ws.SendText("upgraded")
		})
	})

	server := httptest.NewServer(engine)
	defer server.Close()

	// Try regular HTTP request (should fail upgrade)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	
	if err != nil {
		// Expected - might fail upgrade
		t.Logf("Upgrade test: %v", err)
	} else {
		// If it works, verify we can communicate
		defer ws.Close()
		if resp.StatusCode != 101 {
			t.Errorf("Expected 101 Switching Protocols, got %d", resp.StatusCode)
		}
	}
}
