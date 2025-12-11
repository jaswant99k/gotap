package goTap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResolveAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"No address", []string{}, ":5066"},
		{"Single address", []string{":8080"}, ":8080"},
		{"Custom address", []string{"localhost:3000"}, "localhost:3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveAddress(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveAddressPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for too many parameters")
		}
	}()
	resolveAddress([]string{":8080", ":9090"})
}

func TestEngineShutdown(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Start server in background
	server := &http.Server{
		Addr:    ":0", // Random port
		Handler: engine,
	}

	go func() {
		server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestEngineShutdownWithTimeout(t *testing.T) {
	engine := New()
	engine.GET("/slow", func(c *Context) {
		time.Sleep(2 * time.Second)
		c.String(200, "done")
	})

	server := &http.Server{
		Addr:    ":0",
		Handler: engine,
	}

	go func() {
		server.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	// Shutdown with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This may return an error due to timeout, which is expected
	server.Shutdown(ctx)
}

func TestResponseWriterHijack(t *testing.T) {
	engine := New()
	engine.GET("/hijack", func(c *Context) {
		// Test Hijack interface
		if hijacker, ok := c.Writer.(http.Hijacker); ok {
			// Just check that the interface is satisfied
			// Actual hijacking would require a real network connection
			_ = hijacker
			c.String(200, "hijacker available")
		} else {
			t.Error("ResponseWriter should implement http.Hijacker")
			c.String(500, "no hijacker")
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hijack", nil)
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestResponseWriterFlush(t *testing.T) {
	engine := New()
	flushed := false

	engine.GET("/flush", func(c *Context) {
		// Test Flush interface
		if flusher, ok := c.Writer.(http.Flusher); ok {
			// Flush should not panic
			flusher.Flush()
			flushed = true
			c.String(200, "flushed")
		} else {
			t.Error("ResponseWriter should implement http.Flusher")
			c.String(500, "no flusher")
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/flush", nil)
	engine.ServeHTTP(w, req)

	if !flushed {
		t.Error("Flush was not called")
	}
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
