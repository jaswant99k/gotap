package goTap

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	router := New()

	// Setup BasicAuth middleware with test accounts
	authorized := router.Group("/admin", BasicAuth(Accounts{
		"admin": "secret",
		"user1": "password1",
		"user2": "password2",
		"测试用户":  "测试密码", // Test Unicode support
	}))

	authorized.GET("/dashboard", func(c *Context) {
		user, _ := c.Get("user")
		c.JSON(http.StatusOK, H{
			"message": "Welcome to dashboard",
			"user":    user,
		})
	})

	// Test 1: Without credentials - should fail
	t.Run("Without credentials", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}

		authHeader := w.Header().Get("WWW-Authenticate")
		if authHeader != `Basic realm="Authorization Required"` {
			t.Errorf("Expected WWW-Authenticate header, got: %s", authHeader)
		}
	})

	// Test 2: With valid credentials - should succeed
	t.Run("With valid credentials", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
		// Basic auth: admin:secret
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if !stringContains(body, "admin") {
			t.Error("Expected user 'admin' in response")
		}
	})

	// Test 3: With invalid username - should fail
	t.Run("With invalid username", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("hacker:secret")))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	// Test 4: With invalid password - should fail
	t.Run("With invalid password", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword")))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	// Test 5: With Unicode credentials
	t.Run("With Unicode credentials", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("测试用户:测试密码")))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for Unicode credentials, got %d", w.Code)
		}
	})

	// Test 6: Multiple users
	t.Run("Multiple users", func(t *testing.T) {
		users := []struct {
			username string
			password string
		}{
			{"admin", "secret"},
			{"user1", "password1"},
			{"user2", "password2"},
		}

		for _, u := range users {
			req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
			req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(u.username+":"+u.password)))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for user %s, got %d", u.username, w.Code)
			}
		}
	})
}

func TestBasicAuthForRealm(t *testing.T) {
	router := New()

	// Custom realm
	authorized := router.Group("/api", BasicAuthForRealm(Accounts{
		"api_user": "api_key",
	}, "POS Terminal API"))

	authorized.GET("/data", func(c *Context) {
		c.JSON(http.StatusOK, H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/api/data", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	authHeader := w.Header().Get("WWW-Authenticate")
	expected := `Basic realm="POS Terminal API"`
	if authHeader != expected {
		t.Errorf("Expected realm '%s', got: %s", expected, authHeader)
	}
}

func TestBasicAuthMalformedHeader(t *testing.T) {
	router := New()

	authorized := router.Group("/secure", BasicAuth(Accounts{
		"test": "pass",
	}))

	authorized.GET("/data", func(c *Context) {
		c.String(http.StatusOK, "OK")
	})

	testCases := []struct {
		name   string
		header string
	}{
		{"Missing Basic prefix", "admin:secret"},
		{"Invalid base64", "Basic !!!invalid!!!"},
		{"No colon separator", "Basic " + base64.StdEncoding.EncodeToString([]byte("adminnoseparator"))},
		{"Empty authorization", ""},
		{"Only username", "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:"))},
		{"Only password", "Basic " + base64.StdEncoding.EncodeToString([]byte(":password"))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/secure/data", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s: Expected status 401, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestBasicAuthContext(t *testing.T) {
	router := New()

	authorized := router.Group("/", BasicAuth(Accounts{
		"john": "doe",
	}))

	// Test that user is set in context
	authorized.GET("/user", func(c *Context) {
		user, exists := c.Get("user")
		if !exists {
			t.Error("User should be set in context")
		}

		username, ok := user.(string)
		if !ok {
			t.Error("User should be a string")
		}

		if username != "john" {
			t.Errorf("Expected username 'john', got '%s'", username)
		}

		c.String(http.StatusOK, "User: "+username)
	})

	req, _ := http.NewRequest("GET", "/user", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("john:doe")))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "User: john" {
		t.Errorf("Expected 'User: john', got: %s", w.Body.String())
	}
}

func TestBasicAuthSecurityTimingAttack(t *testing.T) {
	// This test ensures we use constant-time comparison
	// While we can't truly test timing, we verify the function works correctly

	router := New()
	authorized := router.Group("/", BasicAuth(Accounts{
		"user": "pass123456789", // Long password
	}))

	authorized.GET("/test", func(c *Context) {
		c.String(http.StatusOK, "OK")
	})

	// Test with passwords of different lengths
	testPasswords := []string{
		"p",              // Very short
		"pass",           // Short
		"pass12345678",   // Almost correct
		"pass123456789",  // Correct
		"pass1234567890", // Longer than correct
	}

	for _, password := range testPasswords {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:"+password)))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if password == "pass123456789" {
			if w.Code != http.StatusOK {
				t.Errorf("Correct password should succeed, got %d", w.Code)
			}
		} else {
			if w.Code != http.StatusUnauthorized {
				t.Errorf("Wrong password '%s' should fail, got %d", password, w.Code)
			}
		}
	}
}

// Benchmark BasicAuth
func BenchmarkBasicAuth(b *testing.B) {
	router := New()
	authorized := router.Group("/", BasicAuth(Accounts{
		"user1": "password1",
		"user2": "password2",
		"user3": "password3",
	}))

	authorized.GET("/test", func(c *Context) {
		c.String(http.StatusOK, "OK")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user2:password2")))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// Helper function
func stringContains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
