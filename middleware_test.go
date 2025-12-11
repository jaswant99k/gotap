// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTAuth(t *testing.T) {
	secret := "test-secret"

	// Generate a valid token
	claims := JWTClaims{
		UserID:    "user123",
		Username:  "testuser",
		Role:      "admin",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	token, err := GenerateJWT(secret, claims)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	r := New()
	r.Use(JWTAuth(secret))

	r.GET("/protected", func(c *Context) {
		retrievedClaims, exists := GetJWTClaims(c)
		if !exists {
			t.Error("JWT claims not found in context")
		}
		if retrievedClaims.UserID != "user123" {
			t.Errorf("Expected UserID 'user123', got '%s'", retrievedClaims.UserID)
		}
		c.JSON(200, H{"status": "ok"})
	})

	// Test with valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test without token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 without token, got %d", w.Code)
	}

	// Test with invalid token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 with invalid token, got %d", w.Code)
	}
}

func TestTransactionID(t *testing.T) {
	r := New()
	r.Use(TransactionID())

	r.GET("/test", func(c *Context) {
		txID := GetTransactionID(c)
		if txID == "" {
			t.Error("Transaction ID should not be empty")
		}
		c.String(200, txID)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Check if transaction ID is in response header
	txID := w.Header().Get("X-Transaction-ID")
	if txID == "" {
		t.Error("Transaction ID not found in response header")
	}

	// Check if it's in response body
	if w.Body.String() != txID {
		t.Errorf("Transaction ID mismatch: header=%s, body=%s", txID, w.Body.String())
	}
}

func TestRateLimiter(t *testing.T) {
	r := New()
	r.Use(RateLimiter(2, 1*time.Second)) // 2 requests per second

	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// First request should succeed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for first request, got %d", w.Code)
	}

	// Second request should succeed
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for second request, got %d", w.Code)
	}

	// Third request should be rate limited
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(w, req)

	if w.Code != 429 {
		t.Errorf("Expected status 429 for third request (rate limited), got %d", w.Code)
	}

	// Different IP should not be rate limited
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for different IP, got %d", w.Code)
	}
}

func TestIPWhitelist(t *testing.T) {
	r := New()
	r.Use(IPWhitelist("127.0.0.1", "192.168.1.0/24"))

	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// Test allowed IP (exact match)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for whitelisted IP, got %d", w.Code)
	}

	// Test allowed IP (CIDR range)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for IP in CIDR range, got %d", w.Code)
	}

	// Test blocked IP
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403 for non-whitelisted IP, got %d", w.Code)
	}
}

func TestRequireRole(t *testing.T) {
	secret := "test-secret"

	r := New()
	r.Use(JWTAuth(secret))

	// Admin-only route
	r.GET("/admin", RequireRole("admin"), func(c *Context) {
		c.String(200, "admin access")
	})

	// Generate admin token
	adminClaims := JWTClaims{
		UserID:    "admin123",
		Role:      "admin",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	adminToken, _ := GenerateJWT(secret, adminClaims)

	// Test with admin token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for admin role, got %d", w.Code)
	}

	// Generate user token (non-admin)
	userClaims := JWTClaims{
		UserID:    "user123",
		Role:      "user",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	userToken, _ := GenerateJWT(secret, userClaims)

	// Test with user token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403 for non-admin role, got %d", w.Code)
	}
}

func BenchmarkJWTAuth(b *testing.B) {
	secret := "test-secret"
	claims := JWTClaims{
		UserID:    "user123",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	token, _ := GenerateJWT(secret, claims)

	r := New()
	r.Use(JWTAuth(secret))
	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}
