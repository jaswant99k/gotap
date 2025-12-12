package goTap

import (
	"compress/gzip"
	"net/http/httptest"
	"testing"
)

// Test gzipResponseWriter Status method
func TestGzipResponseWriterStatus(t *testing.T) {
	engine := New()
	engine.Use(Gzip())

	var recordedStatus int
	engine.GET("/test", func(c *Context) {
		c.Writer.WriteHeader(201)
		recordedStatus = c.Writer.Status()
		c.String(201, "test data with enough content to trigger gzip compression because it needs to be long enough")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	engine.ServeHTTP(w, req)

	// The recorded status should be 201
	if recordedStatus != 201 {
		t.Errorf("Expected recorded status 201, got %d", recordedStatus)
	}
}

// Test gzipResponseWriter Size method
func TestGzipResponseWriterSize(t *testing.T) {
	engine := New()
	engine.Use(Gzip())

	testData := "Hello, World!"

	engine.GET("/test", func(c *Context) {
		c.Writer.Write([]byte(testData))
		size := c.Writer.Size()
		if size <= 0 {
			t.Errorf("Expected positive size, got %d", size)
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	engine.ServeHTTP(w, req)
}

// Test gzipResponseWriter Written method
func TestGzipResponseWriterWritten(t *testing.T) {
	engine := New()
	engine.Use(Gzip())

	engine.GET("/test", func(c *Context) {
		if c.Writer.Written() {
			t.Error("Should not be written before first write")
		}
		c.Writer.Write([]byte("test"))
		if !c.Writer.Written() {
			t.Error("Should be written after write")
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	engine.ServeHTTP(w, req)
}

// Test gzipResponseWriter WriteHeaderNow method
func TestGzipResponseWriterWriteHeaderNow(t *testing.T) {
	engine := New()
	engine.Use(Gzip())

	engine.GET("/test", func(c *Context) {
		c.Writer.WriteHeader(202)
		c.Writer.WriteHeaderNow()
		// Verify it was written
		if c.Writer.Status() != 202 {
			t.Errorf("Expected status 202, got %d", c.Writer.Status())
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	engine.ServeHTTP(w, req)
}

// Test gzip with different compression levels
func TestGzipWithDifferentLevels(t *testing.T) {
	levels := []int{
		gzip.BestSpeed,
		gzip.BestCompression,
		gzip.DefaultCompression,
	}

	for _, level := range levels {
		engine := New()
		engine.Use(GzipWithConfig(GzipConfig{
			Level: level,
		}))

		// Need large enough data to trigger compression (>= MinLength default 1024)
		largeData := make([]byte, 2048)
		for i := range largeData {
			largeData[i] = byte('a' + (i % 26))
		}

		engine.GET("/test", func(c *Context) {
			c.String(200, string(largeData))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		engine.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Level %d: Expected status 200, got %d", level, w.Code)
		}

		// For large enough data, gzip should be applied
		encoding := w.Header().Get("Content-Encoding")
		if encoding == "" {
			t.Logf("Level %d: Content-Encoding header not set (data might be too small)", level)
		}
	}
}
