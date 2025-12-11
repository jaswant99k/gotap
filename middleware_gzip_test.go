package goTap

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzip(t *testing.T) {
	t.Run("Default config compresses large responses", func(t *testing.T) {
		router := New()
		router.Use(Gzip())
		router.GET("/test", func(c *Context) {
			// Send a large response (>1KB)
			data := strings.Repeat("Hello, World! ", 100) // ~1.3KB
			c.String(http.StatusOK, data)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Expected Content-Encoding: gzip")
		}

		if w.Header().Get("Vary") != "Accept-Encoding" {
			t.Error("Expected Vary: Accept-Encoding")
		}

		// Decompress and verify content
		gr, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		if err != nil {
			t.Fatalf("Failed to decompress: %v", err)
		}

		expected := strings.Repeat("Hello, World! ", 100)
		if string(decompressed) != expected {
			t.Error("Decompressed content doesn't match original")
		}
	})

	t.Run("Small responses not compressed", func(t *testing.T) {
		router := New()
		router.Use(Gzip())
		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, "Small response") // <1KB
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Small response should NOT be compressed
		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("Small response should not be compressed")
		}

		if w.Body.String() != "Small response" {
			t.Error("Response body doesn't match expected")
		}
	})

	t.Run("No compression without Accept-Encoding", func(t *testing.T) {
		router := New()
		router.Use(Gzip())
		router.GET("/test", func(c *Context) {
			data := strings.Repeat("Hello, World! ", 100)
			c.String(http.StatusOK, data)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No Accept-Encoding header
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("Should not compress without Accept-Encoding")
		}
	})
}

func TestGzipWithConfig(t *testing.T) {
	t.Run("Custom compression level", func(t *testing.T) {
		router := New()
		router.Use(GzipWithConfig(GzipConfig{
			Level:     gzip.BestCompression,
			MinLength: 100,
		}))
		router.GET("/test", func(c *Context) {
			data := strings.Repeat("A", 500) // 500 bytes
			c.String(http.StatusOK, data)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Expected Content-Encoding: gzip")
		}

		// Verify it's compressed
		gr, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		if err != nil {
			t.Fatalf("Failed to decompress: %v", err)
		}

		if string(decompressed) != strings.Repeat("A", 500) {
			t.Error("Decompressed content doesn't match")
		}
	})

	t.Run("Custom MinLength threshold", func(t *testing.T) {
		router := New()
		router.Use(GzipWithConfig(GzipConfig{
			Level:     gzip.DefaultCompression,
			MinLength: 2048, // 2KB minimum
		}))
		router.GET("/test", func(c *Context) {
			data := strings.Repeat("Hello, World! ", 100) // ~1.3KB
			c.String(http.StatusOK, data)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Response is 1.3KB but MinLength is 2KB, so should NOT compress
		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("Should not compress below MinLength threshold")
		}
	})

	t.Run("Excluded extensions", func(t *testing.T) {
		router := New()
		router.Use(GzipWithConfig(GzipConfig{
			Level:              gzip.DefaultCompression,
			MinLength:          100,
			ExcludedExtensions: []string{".jpg", ".png", ".zip"},
		}))
		router.GET("/image.jpg", func(c *Context) {
			data := strings.Repeat("Image data ", 200)
			c.String(http.StatusOK, data)
		})

		req := httptest.NewRequest(http.MethodGet, "/image.jpg", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("Should not compress excluded extension .jpg")
		}
	})

	t.Run("Excluded paths", func(t *testing.T) {
		router := New()
		router.Use(GzipWithConfig(GzipConfig{
			Level:         gzip.DefaultCompression,
			MinLength:     100,
			ExcludedPaths: []string{"/api/download", "/static"},
		}))
		router.GET("/api/download/file", func(c *Context) {
			data := strings.Repeat("File content ", 200)
			c.String(http.StatusOK, data)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/download/file", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("Should not compress excluded path /api/download")
		}
	})
}

func TestGzipJSON(t *testing.T) {
	t.Run("Compress JSON responses", func(t *testing.T) {
		router := New()
		router.Use(Gzip())
		router.GET("/api/data", func(c *Context) {
			// Large JSON response
			data := make([]map[string]interface{}, 100)
			for i := 0; i < 100; i++ {
				data[i] = map[string]interface{}{
					"id":   i,
					"name": "Item " + string(rune(i)),
					"desc": "This is a description for item number " + string(rune(i)),
				}
			}
			c.JSON(http.StatusOK, data)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Expected Content-Encoding: gzip")
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		// Verify it's valid gzip
		gr, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		if err != nil {
			t.Fatalf("Failed to decompress: %v", err)
		}

		if len(decompressed) == 0 {
			t.Error("Decompressed JSON is empty")
		}
	})
}

func TestGzipPOSExample(t *testing.T) {
	t.Run("POS API response compression", func(t *testing.T) {
		router := New()

		// Configure Gzip for POS API
		router.Use(GzipWithConfig(GzipConfig{
			Level:     gzip.BestSpeed, // Fast compression for real-time responses
			MinLength: 512,            // Compress responses >512 bytes
			ExcludedExtensions: []string{
				".jpg", ".png", ".pdf", // Don't compress receipts/images
			},
			ExcludedPaths: []string{
				"/api/receipt/download", // Don't compress receipt downloads
			},
		}))

		// Large transaction history endpoint
		router.GET("/api/transactions", func(c *Context) {
			transactions := make([]map[string]interface{}, 50)
			for i := 0; i < 50; i++ {
				transactions[i] = map[string]interface{}{
					"transaction_id": "TXN-" + string(rune(1000+i)),
					"amount":         99.99,
					"items":          []string{"Item A", "Item B", "Item C"},
					"timestamp":      "2024-01-15T10:30:00Z",
				}
			}
			c.JSON(http.StatusOK, transactions)
		})

		// Receipt download endpoint (should NOT compress)
		router.GET("/api/receipt/download", func(c *Context) {
			receipt := strings.Repeat("Receipt data\n", 100)
			c.String(http.StatusOK, receipt)
		})

		// Test transaction endpoint (should compress)
		t.Run("Compress transaction list", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if w.Header().Get("Content-Encoding") != "gzip" {
				t.Error("Transaction list should be compressed")
			}

			// Verify decompression works
			gr, err := gzip.NewReader(w.Body)
			if err != nil {
				t.Fatalf("Failed to decompress: %v", err)
			}
			defer gr.Close()

			data, err := io.ReadAll(gr)
			if err != nil {
				t.Fatalf("Failed to read decompressed data: %v", err)
			}

			if len(data) == 0 {
				t.Error("Decompressed data is empty")
			}
		})

		// Test receipt endpoint (should NOT compress due to excluded path)
		t.Run("Exclude receipt download", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/receipt/download", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Header().Get("Content-Encoding") == "gzip" {
				t.Error("Receipt download should not be compressed")
			}
		})
	})
}

func TestGzipCompressionRatio(t *testing.T) {
	t.Run("Verify compression reduces size", func(t *testing.T) {
		router := New()
		router.Use(Gzip())

		// Highly compressible data
		compressibleData := strings.Repeat("A", 10000) // 10KB of same character

		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, compressibleData)
		})

		// Test WITH compression
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.Header.Set("Accept-Encoding", "gzip")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		compressedSize := w1.Body.Len()

		// Test WITHOUT compression
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		uncompressedSize := w2.Body.Len()

		if compressedSize >= uncompressedSize {
			t.Errorf("Compressed size (%d) should be smaller than uncompressed (%d)",
				compressedSize, uncompressedSize)
		}

		compressionRatio := float64(uncompressedSize) / float64(compressedSize)
		t.Logf("Compression ratio: %.2fx (uncompressed: %d bytes, compressed: %d bytes)",
			compressionRatio, uncompressedSize, compressedSize)

		// Expect at least 2x compression for this highly repetitive data
		if compressionRatio < 2.0 {
			t.Errorf("Expected at least 2x compression, got %.2fx", compressionRatio)
		}
	})
}

// Benchmarks
func BenchmarkGzip(b *testing.B) {
	router := New()
	router.Use(Gzip())
	router.GET("/test", func(c *Context) {
		data := strings.Repeat("Hello, World! ", 100)
		c.String(http.StatusOK, data)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGzipWithConfig(b *testing.B) {
	router := New()
	router.Use(GzipWithConfig(GzipConfig{
		Level:     gzip.BestSpeed,
		MinLength: 512,
	}))
	router.GET("/test", func(c *Context) {
		data := strings.Repeat("Hello, World! ", 100)
		c.String(http.StatusOK, data)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkNoGzip(b *testing.B) {
	router := New()
	router.GET("/test", func(c *Context) {
		data := strings.Repeat("Hello, World! ", 100)
		c.String(http.StatusOK, data)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
