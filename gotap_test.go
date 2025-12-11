// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEngine(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}

	if r.RouterGroup.engine != r {
		t.Error("Engine not properly initialized")
	}
}

func TestDefault(t *testing.T) {
	r := Default()
	if r == nil {
		t.Fatal("Default() returned nil")
	}

	if len(r.Handlers) != 2 {
		t.Errorf("Expected 2 default handlers (Logger, Recovery), got %d", len(r.Handlers))
	}
}

func TestSimpleRoute(t *testing.T) {
	r := New()

	r.GET("/test", func(c *Context) {
		c.String(200, "test response")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got '%s'", w.Body.String())
	}
}

func TestRouteWithParams(t *testing.T) {
	r := New()

	r.GET("/user/:id", func(c *Context) {
		id := c.Param("id")
		c.String(200, "user %s", id)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "user 123" {
		t.Errorf("Expected 'user 123', got '%s'", w.Body.String())
	}
}

func TestWildcardRoute(t *testing.T) {
	// t.Skip("Wildcard route needs debugging - known issue")
	r := New()

	r.GET("/files/*filepath", func(c *Context) {
		path := c.Param("filepath")
		c.String(200, "filepath: %s", path)
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/files/test.txt", "filepath: /test.txt"},
		{"/files/dir/file.txt", "filepath: /dir/file.txt"},
		{"/files/", "filepath: /"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", tt.path, nil)
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected '%s' for %s, got '%s'", tt.expected, tt.path, w.Body.String())
		}
	}
}

func TestRouterGroup(t *testing.T) {
	r := New()

	v1 := r.Group("/v1")
	v1.GET("/users", func(c *Context) {
		c.String(200, "v1 users")
	})

	v2 := r.Group("/v2")
	v2.GET("/users", func(c *Context) {
		c.String(200, "v2 users")
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/v1/users", "v1 users"},
		{"/v2/users", "v2 users"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", tt.path, nil)
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", tt.path, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("Expected '%s' for %s, got '%s'", tt.expected, tt.path, w.Body.String())
		}
	}
}

func TestMiddleware(t *testing.T) {
	r := New()

	executed := false
	r.Use(func(c *Context) {
		executed = true
		c.Next()
	})

	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if !executed {
		t.Error("Middleware was not executed")
	}
}

func TestAbort(t *testing.T) {
	r := New()

	handlerExecuted := false

	r.Use(func(c *Context) {
		c.Abort()
	})

	r.GET("/test", func(c *Context) {
		handlerExecuted = true
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if handlerExecuted {
		t.Error("Handler should not have been executed after Abort()")
	}
}

func TestHTTPMethods(t *testing.T) {
	r := New()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, method := range methods {
		r.Handle(method, "/test", func(c *Context) {
			c.String(200, method)
		})
	}

	for _, method := range methods {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(method, "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for %s, got %d", method, w.Code)
		}

		if method != "HEAD" && w.Body.String() != method {
			t.Errorf("Expected '%s' for %s, got '%s'", method, method, w.Body.String())
		}
	}
}
