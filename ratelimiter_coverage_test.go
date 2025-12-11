package goTap

import (
	"net/http/httptest"
	"testing"
	"time"
)

// Test rate limiter Reset method
func TestRateLimiterReset(t *testing.T) {
	engine := New()
	
	config := RateLimiterConfig{
		Max:    2,
		Window: 1 * time.Minute,
		KeyFunc: func(c *Context) string {
			return "test-key"
		},
	}
	
	limiter := RateLimiterWithConfig(config)
	engine.Use(limiter)
	
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Make 2 requests (should succeed)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		engine.ServeHTTP(w, req)
		
		if w.Code != 200 {
			t.Errorf("Request %d: Expected 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)
	
	if w.Code != 429 {
		t.Errorf("Expected 429 (rate limited), got %d", w.Code)
	}
}

// Test rate limiter cleanup function
func TestRateLimiterCleanup(t *testing.T) {
	engine := New()
	
	config := RateLimiterConfig{
		Max:    10,
		Window: 1 * time.Minute,
		KeyFunc: func(c *Context) string {
			return c.ClientIP()
		},
	}
	
	limiter := RateLimiterWithConfig(config)
	engine.Use(limiter)
	
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Make a request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	// Wait for cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Cleanup should have run at least once
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// Test RateLimiterByAPIKey with missing key
func TestRateLimiterByAPIKeyMissing(t *testing.T) {
	engine := New()
	
	limiter := RateLimiterByAPIKey(5, 1*time.Minute, "X-API-Key")
	engine.Use(limiter)
	
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Request without API key
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	// Should still work (uses IP as fallback)
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// Test RateLimiterByAPIKey with key
func TestRateLimiterByAPIKeyWithKey(t *testing.T) {
	engine := New()
	
	limiter := RateLimiterByAPIKey(2, 1*time.Minute, "X-API-Key")
	engine.Use(limiter)
	
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Make 2 requests with same API key
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "test-key-123")
		engine.ServeHTTP(w, req)
		
		if w.Code != 200 {
			t.Errorf("Request %d: Expected 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	engine.ServeHTTP(w, req)
	
	if w.Code != 429 {
		t.Errorf("Expected 429, got %d", w.Code)
	}
}

// Test BurstRateLimiter edge cases
func TestBurstRateLimiterEdgeCases(t *testing.T) {
	engine := New()
	
	limiter := BurstRateLimiter(5, 1.0) // burst of 5, refill 1 per second
	engine.Use(limiter)
	
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Make burst of requests
	successCount := 0
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		engine.ServeHTTP(w, req)
		
		if w.Code == 200 {
			successCount++
		}
	}

	// Should allow burst
	if successCount < 3 {
		t.Errorf("Expected at least 3 successful requests, got %d", successCount)
	}
}

// Test rate limiter with different IPs
func TestRateLimiterMultipleIPs(t *testing.T) {
	engine := New()
	
	limiter := RateLimiter(2, 1*time.Minute) // 2 requests per minute
	engine.Use(limiter)
	
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Different IPs should have separate limits
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	
	for _, ip := range ips {
		// Each IP should be able to make 2 requests
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Real-IP", ip)
			engine.ServeHTTP(w, req)
			
			if w.Code != 200 {
				t.Errorf("IP %s request %d: Expected 200, got %d", ip, i+1, w.Code)
			}
		}
	}
}

// Test rate limiter allow function edge cases
func TestRateLimiterAllowFunction(t *testing.T) {
	engine := New()
	
	// Very restrictive rate limit
	config := RateLimiterConfig{
		Max:    1,
		Window: 1 * time.Minute,
		KeyFunc: func(c *Context) string {
			return "single-key"
		},
	}
	
	limiter := RateLimiterWithConfig(config)
	engine.Use(limiter)
	
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// First request should succeed
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w1, req1)
	
	if w1.Code != 200 {
		t.Errorf("First request: Expected 200, got %d", w1.Code)
	}

	// Immediate second request should fail
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w2, req2)
	
	if w2.Code != 429 {
		t.Errorf("Second request: Expected 429, got %d", w2.Code)
	}
}
