package goTap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCORS(t *testing.T) {
	router := New()
	router.Use(CORS())

	router.GET("/test", func(c *Context) {
		c.String(http.StatusOK, "OK")
	})

	// Test 1: Simple request with default config
	t.Run("Default config allows all origins", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", allowOrigin)
		}
	})

	// Test 2: Preflight OPTIONS request
	t.Run("Preflight OPTIONS request", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		allowMethods := w.Header().Get("Access-Control-Allow-Methods")
		if allowMethods == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}

		allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
		if allowHeaders == "" {
			t.Error("Expected Access-Control-Allow-Headers header")
		}
	})
}

func TestCORSWithConfig(t *testing.T) {
	// Test 3: Specific origins only
	t.Run("Whitelist specific origins", func(t *testing.T) {
		router := New()
		router.Use(CORSWithConfig(CORSConfig{
			AllowOrigins: []string{"http://example.com", "https://api.example.com"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type"},
		}))

		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, "OK")
		})

		// Allowed origin
		req1, _ := http.NewRequest("GET", "/test", nil)
		req1.Header.Set("Origin", "http://example.com")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		allowOrigin1 := w1.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin1 != "http://example.com" {
			t.Errorf("Expected origin http://example.com, got %s", allowOrigin1)
		}

		// Disallowed origin
		req2, _ := http.NewRequest("GET", "/test", nil)
		req2.Header.Set("Origin", "http://evil.com")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		allowOrigin2 := w2.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin2 != "" {
			t.Errorf("Expected no CORS header for disallowed origin, got %s", allowOrigin2)
		}
	})

	// Test 4: Credentials
	t.Run("Allow credentials", func(t *testing.T) {
		router := New()
		router.Use(CORSWithConfig(CORSConfig{
			AllowOrigins:     []string{"http://example.com"},
			AllowCredentials: true,
		}))

		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		credentials := w.Header().Get("Access-Control-Allow-Credentials")
		if credentials != "true" {
			t.Errorf("Expected Access-Control-Allow-Credentials: true, got %s", credentials)
		}
	})

	// Test 5: Expose headers
	t.Run("Expose headers", func(t *testing.T) {
		router := New()
		router.Use(CORSWithConfig(CORSConfig{
			AllowOrigins:  []string{"*"},
			ExposeHeaders: []string{"X-Custom-Header", "X-Another-Header"},
		}))

		router.GET("/test", func(c *Context) {
			c.Header("X-Custom-Header", "value")
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		exposeHeaders := w.Header().Get("Access-Control-Expose-Headers")
		if exposeHeaders != "X-Custom-Header, X-Another-Header" {
			t.Errorf("Expected exposed headers, got %s", exposeHeaders)
		}
	})

	// Test 6: Max age
	t.Run("Max age for preflight", func(t *testing.T) {
		router := New()
		router.Use(CORSWithConfig(CORSConfig{
			AllowOrigins: []string{"*"},
			MaxAge:       24 * time.Hour,
		}))

		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		maxAge := w.Header().Get("Access-Control-Max-Age")
		expected := "86400" // 24 hours in seconds
		if maxAge != expected {
			t.Errorf("Expected Max-Age %s, got %s", expected, maxAge)
		}
	})
}

func TestCORSWildcard(t *testing.T) {
	// Test 7: Wildcard subdomain matching
	t.Run("Wildcard subdomain", func(t *testing.T) {
		router := New()
		router.Use(CORSWithConfig(CORSConfig{
			AllowOrigins:  []string{"https://*.example.com"},
			AllowWildcard: true,
		}))

		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, "OK")
		})

		testCases := []struct {
			origin   string
			expected bool
		}{
			{"https://api.example.com", true},
			{"https://app.example.com", true},
			{"https://test.example.com", true},
			{"https://example.com", false},    // No subdomain
			{"http://api.example.com", false}, // HTTP not HTTPS
			{"https://example.org", false},    // Different domain
		}

		for _, tc := range testCases {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			hasHeader := allowOrigin != ""

			if hasHeader != tc.expected {
				t.Errorf("Origin %s: expected allowed=%v, got allowed=%v", tc.origin, tc.expected, hasHeader)
			}

			if tc.expected && allowOrigin != tc.origin {
				t.Errorf("Expected origin %s, got %s", tc.origin, allowOrigin)
			}
		}
	})
}

func TestCORSWithCustomFunc(t *testing.T) {
	// Test 8: Custom origin validation function
	t.Run("Custom allow origin function", func(t *testing.T) {
		router := New()
		router.Use(CORSWithConfig(CORSConfig{
			AllowOriginFunc: func(origin string) bool {
				// Only allow origins ending with .trusted.com
				return len(origin) > 12 && origin[len(origin)-12:] == ".trusted.com"
			},
		}))

		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, "OK")
		})

		testCases := []struct {
			origin   string
			expected bool
		}{
			{"https://api.trusted.com", true},
			{"https://app.trusted.com", true},
			{"https://evil.com", false},
			{"https://trusted.com", false}, // Doesn't end with .trusted.com
		}

		for _, tc := range testCases {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			hasHeader := allowOrigin != ""

			if hasHeader != tc.expected {
				t.Errorf("Origin %s: expected allowed=%v, got allowed=%v", tc.origin, tc.expected, hasHeader)
			}
		}
	})
}

func TestCORSNoOriginHeader(t *testing.T) {
	// Test 9: Request without Origin header
	t.Run("No Origin header", func(t *testing.T) {
		router := New()
		router.Use(CORS())

		router.GET("/test", func(c *Context) {
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		// No Origin header set
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Should still have CORS header for * config
		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("Expected *, got %s", allowOrigin)
		}
	})
}

func TestCORSPOSExample(t *testing.T) {
	// Test 10: POS terminal CORS configuration
	t.Run("POS terminal CORS setup", func(t *testing.T) {
		router := New()

		// Only allow specific POS terminal origins
		router.Use(CORSWithConfig(CORSConfig{
			AllowOrigins:     []string{"https://pos.retailer.com", "https://terminal.retailer.com"},
			AllowMethods:     []string{"GET", "POST", "PUT"},
			AllowHeaders:     []string{"Content-Type", "Authorization", "X-Transaction-ID"},
			ExposeHeaders:    []string{"X-Transaction-ID", "X-Receipt-Number"},
			AllowCredentials: true,
			MaxAge:           1 * time.Hour,
		}))

		router.POST("/api/transaction", func(c *Context) {
			c.Header("X-Transaction-ID", "TXN-12345")
			c.Header("X-Receipt-Number", "RCP-67890")
			c.JSON(http.StatusOK, H{"status": "success"})
		})

		req, _ := http.NewRequest("POST", "/api/transaction", nil)
		req.Header.Set("Origin", "https://pos.retailer.com")
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "https://pos.retailer.com" {
			t.Error("Origin not allowed")
		}

		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("Credentials not allowed")
		}

		exposeHeaders := w.Header().Get("Access-Control-Expose-Headers")
		if exposeHeaders != "X-Transaction-ID, X-Receipt-Number" {
			t.Errorf("Exposed headers incorrect: %s", exposeHeaders)
		}
	})
}
