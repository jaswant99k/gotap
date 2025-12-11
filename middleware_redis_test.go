package goTap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupMiniRedis creates an in-memory Redis server for testing
func setupMiniRedis(t *testing.T) (*RedisClient, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	redisClient := &RedisClient{
		Client: client,
		ctx:    context.Background(),
	}

	return redisClient, mr
}

func TestNewRedisClient(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client, err := NewRedisClient(mr.Addr(), "", 0)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	if client.Client == nil {
		t.Error("Redis client is nil")
	}

	// Test ping
	ctx := context.Background()
	if err := client.Client.Ping(ctx).Err(); err != nil {
		t.Errorf("Redis ping failed: %v", err)
	}
}

func TestNewRedisClientFailure(t *testing.T) {
	// Try to connect to non-existent Redis server
	_, err := NewRedisClient("localhost:9999", "", 0)
	if err == nil {
		t.Error("Expected error when connecting to non-existent Redis server")
	}
}

func TestRedisCache(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisCache(RedisCacheConfig{
		Client: redisClient,
		TTL:    1 * time.Second,
	}))

	counter := 0
	r.GET("/test", func(c *Context) {
		counter++
		c.JSON(200, H{"count": counter})
	})

	// First request - cache miss
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w1, req1)

	if w1.Code != 200 {
		t.Errorf("Expected status 200, got %d", w1.Code)
	}

	cacheHeader1 := w1.Header().Get("X-Cache")
	if cacheHeader1 != "MISS" {
		t.Errorf("Expected X-Cache: MISS, got %s", cacheHeader1)
	}

	// Second request - cache hit
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	cacheHeader2 := w2.Header().Get("X-Cache")
	if cacheHeader2 != "HIT" {
		t.Errorf("Expected X-Cache: HIT, got %s", cacheHeader2)
	}

	// Counter should still be 1 (served from cache)
	if counter != 1 {
		t.Errorf("Expected counter to be 1, got %d", counter)
	}
}

func TestRedisCacheWithQueryString(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisCache(RedisCacheConfig{
		Client: redisClient,
		TTL:    1 * time.Second,
	}))

	r.GET("/search", func(c *Context) {
		query := c.Query("q")
		c.JSON(200, H{"query": query})
	})

	// Request with query string 1
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/search?q=apple", nil)
	r.ServeHTTP(w1, req1)

	if w1.Header().Get("X-Cache") != "MISS" {
		t.Error("Expected cache miss for first request")
	}

	// Same query string - cache hit
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/search?q=apple", nil)
	r.ServeHTTP(w2, req2)

	if w2.Header().Get("X-Cache") != "HIT" {
		t.Error("Expected cache hit for same query")
	}

	// Different query string - cache miss
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/search?q=orange", nil)
	r.ServeHTTP(w3, req3)

	if w3.Header().Get("X-Cache") != "MISS" {
		t.Error("Expected cache miss for different query")
	}
}

func TestRedisCacheSkipPaths(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisCache(RedisCacheConfig{
		Client:    redisClient,
		TTL:       1 * time.Second,
		SkipPaths: []string{"/admin", "/api/auth"},
	}))

	counter := 0
	r.GET("/admin", func(c *Context) {
		counter++
		c.JSON(200, H{"count": counter})
	})

	// First request
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/admin", nil)
	r.ServeHTTP(w1, req1)

	// Second request - should NOT be cached
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/admin", nil)
	r.ServeHTTP(w2, req2)

	if counter != 2 {
		t.Errorf("Expected counter to be 2 (no caching), got %d", counter)
	}
}

func TestRedisCacheOnlyGET(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisCache(RedisCacheConfig{
		Client: redisClient,
		TTL:    1 * time.Second,
	}))

	counter := 0
	handler := func(c *Context) {
		counter++
		c.JSON(200, H{"count": counter})
	}

	r.GET("/test", handler)
	r.POST("/test", handler)

	// POST request - should NOT be cached
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w2, req2)

	if counter != 2 {
		t.Errorf("Expected counter to be 2 (POST not cached), got %d", counter)
	}
}

func TestRedisCacheCustomKeyGenerator(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisCache(RedisCacheConfig{
		Client: redisClient,
		TTL:    1 * time.Second,
		KeyGenerator: func(c *Context) string {
			// Custom key based on user ID header
			userID := c.GetHeader("X-User-ID")
			return fmt.Sprintf("user:%s:%s", userID, c.Request.URL.Path)
		},
	}))

	counter := make(map[string]int)
	r.GET("/profile", func(c *Context) {
		userID := c.GetHeader("X-User-ID")
		counter[userID]++
		c.JSON(200, H{"user": userID, "count": counter[userID]})
	})

	// User 1 - first request
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/profile", nil)
	req1.Header.Set("X-User-ID", "user1")
	r.ServeHTTP(w1, req1)

	// User 1 - cached
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/profile", nil)
	req2.Header.Set("X-User-ID", "user1")
	r.ServeHTTP(w2, req2)

	if w2.Header().Get("X-Cache") != "HIT" {
		t.Error("Expected cache hit for same user")
	}

	// User 2 - different cache entry
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/profile", nil)
	req3.Header.Set("X-User-ID", "user2")
	r.ServeHTTP(w3, req3)

	if w3.Header().Get("X-Cache") != "MISS" {
		t.Error("Expected cache miss for different user")
	}
}

func TestRedisInjectAndGet(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisInject(redisClient))

	r.GET("/test", func(c *Context) {
		client, ok := GetRedis(c)
		if !ok {
			t.Error("Redis client not found in context")
			c.JSON(500, H{"error": "Redis not available"})
			return
		}

		if client.Client == nil {
			t.Error("Redis client is nil")
		}

		c.JSON(200, H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMustGetRedis(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisInject(redisClient))

	r.GET("/test", func(c *Context) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("MustGetRedis should not panic when Redis is available")
			}
		}()

		client := MustGetRedis(c)
		if client == nil {
			t.Error("Expected Redis client, got nil")
		}
		c.JSON(200, H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestMustGetRedisPanic(t *testing.T) {
	r := New()

	r.GET("/test", func(c *Context) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetRedis should panic when Redis is not available")
			}
		}()

		MustGetRedis(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestRedisSession(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisSession(RedisSessionConfig{
		Client: redisClient,
		TTL:    1 * time.Minute,
	}))

	r.GET("/set", func(c *Context) {
		session, ok := GetSession(c)
		if !ok {
			t.Error("Session not found in context")
			c.JSON(500, H{"error": "Session not available"})
			return
		}

		session.Set("user_id", "12345")
		session.Set("username", "testuser")

		c.JSON(200, H{"status": "session set"})
	})

	r.GET("/get", func(c *Context) {
		session := MustGetSession(c)

		userID, ok := session.Get("user_id")
		if !ok {
			t.Error("user_id not found in session")
		}

		username, ok := session.Get("username")
		if !ok {
			t.Error("username not found in session")
		}

		c.JSON(200, H{
			"user_id":  userID,
			"username": username,
		})
	})

	// Set session values
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/set", nil)
	r.ServeHTTP(w1, req1)

	if w1.Code != 200 {
		t.Errorf("Expected status 200, got %d", w1.Code)
	}

	// Get session cookie
	cookies := w1.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No session cookie set")
	}

	sessionCookie := cookies[0]

	// Get session values with same cookie
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/get", nil)
	req2.AddCookie(sessionCookie)
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}
}

func TestSessionDestroy(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.Use(RedisSession(RedisSessionConfig{
		Client: redisClient,
		TTL:    1 * time.Minute,
	}))

	r.GET("/destroy", func(c *Context) {
		session := MustGetSession(c)
		if err := session.Destroy(); err != nil {
			t.Errorf("Failed to destroy session: %v", err)
		}
		c.JSON(200, H{"status": "destroyed"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/destroy", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRedisHealthCheck(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	r := New()
	r.GET("/health", RedisHealthCheck(redisClient))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !contains(w.Body.String(), "healthy") {
		t.Error("Expected healthy status in response")
	}
}

func TestRedisHealthCheckNoClient(t *testing.T) {
	r := New()
	r.GET("/health", RedisHealthCheck(nil))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 503 {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestRedisPubSub(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Create pub/sub
	pubsub := NewRedisPubSub(redisClient, "test-channel")
	defer pubsub.Close()

	// Subscribe and receive in goroutine
	received := make(chan string, 1)
	go func() {
		for msg := range pubsub.Receive() {
			received <- msg.Payload
			break
		}
	}()

	// Give subscriber time to connect
	time.Sleep(100 * time.Millisecond)

	// Publish message
	err := pubsub.Publish("test-channel", "hello world")
	if err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}

	// Wait for message
	select {
	case msg := <-received:
		if msg != "hello world" {
			t.Errorf("Expected 'hello world', got '%s'", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for pub/sub message")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == "" {
		t.Error("Session ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}

	if len(id1) != 32 {
		t.Errorf("Expected session ID length 32, got %d", len(id1))
	}
}

func TestSessionGetSetDelete(t *testing.T) {
	redisClient, mr := setupMiniRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	session := &Session{
		ID:     "test-session",
		Data:   make(map[string]string),
		client: redisClient,
		key:    "session:test-session",
		ttl:    5 * time.Minute,
	}

	// Test Set
	session.Set("key1", "value1")
	if !session.modified {
		t.Error("Session should be marked as modified")
	}

	// Test Get
	val, ok := session.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got '%s'", val)
	}

	// Test Delete
	session.Delete("key1")
	_, ok = session.Get("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}

	// Test Save
	session.Set("key2", "value2")
	err := session.Save()
	if err != nil {
		t.Errorf("Failed to save session: %v", err)
	}

	if session.modified {
		t.Error("Session should not be modified after save")
	}
}

func TestRedisCacheNilClient(t *testing.T) {
	r := New()
	r.Use(RedisCache(RedisCacheConfig{
		Client: nil, // No client
	}))

	counter := 0
	r.GET("/test", func(c *Context) {
		counter++
		c.JSON(200, H{"count": counter})
	})

	// Should work without caching
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)

	if counter != 2 {
		t.Errorf("Expected counter to be 2 (no caching), got %d", counter)
	}
}
