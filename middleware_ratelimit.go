// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	// Max requests allowed in the time window
	Max int

	// Time window duration
	Window time.Duration

	// KeyFunc defines a function to generate the rate limiter key
	// Default: uses client IP
	KeyFunc func(*Context) string

	// ErrorHandler is called when rate limit is exceeded
	ErrorHandler func(*Context)

	// SkipFunc defines a function to skip rate limiting
	// Useful for whitelisting certain IPs or paths
	SkipFunc func(*Context) bool

	// Store is the storage backend for rate limit data
	// Default: in-memory store
	Store RateLimiterStore
}

// RateLimiterStore defines the interface for rate limiter storage
type RateLimiterStore interface {
	// Increment increments the counter for the given key
	// Returns current count and expiration time
	Increment(key string, window time.Duration) (int, time.Time, error)

	// Reset resets the counter for the given key
	Reset(key string) error
}

// inMemoryStore is a simple in-memory rate limiter store
type inMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	count      int
	expiresAt  time.Time
	windowSize time.Duration
}

func newInMemoryStore() *inMemoryStore {
	store := &inMemoryStore{
		entries: make(map[string]*rateLimitEntry),
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

func (s *inMemoryStore) Increment(key string, window time.Duration) (int, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry, exists := s.entries[key]

	if !exists || now.After(entry.expiresAt) {
		// Create new entry
		entry = &rateLimitEntry{
			count:      1,
			expiresAt:  now.Add(window),
			windowSize: window,
		}
		s.entries[key] = entry
		return 1, entry.expiresAt, nil
	}

	// Increment existing entry
	entry.count++
	return entry.count, entry.expiresAt, nil
}

func (s *inMemoryStore) Reset(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}

func (s *inMemoryStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.entries {
			if now.After(entry.expiresAt) {
				delete(s.entries, key)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimiter returns a rate limiter middleware
// max: maximum requests allowed
// window: time window duration
func RateLimiter(max int, window time.Duration) HandlerFunc {
	return RateLimiterWithConfig(RateLimiterConfig{
		Max:    max,
		Window: window,
	})
}

// RateLimiterWithConfig returns a rate limiter middleware with config
func RateLimiterWithConfig(config RateLimiterConfig) HandlerFunc {
	if config.Max <= 0 {
		panic("rate limiter max must be greater than 0")
	}

	if config.Window <= 0 {
		panic("rate limiter window must be greater than 0")
	}

	if config.KeyFunc == nil {
		config.KeyFunc = func(c *Context) string {
			return c.ClientIP()
		}
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = func(c *Context) {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Max))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(429, H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
		}
	}

	if config.Store == nil {
		config.Store = newInMemoryStore()
	}

	return func(c *Context) {
		// Check if we should skip rate limiting
		if config.SkipFunc != nil && config.SkipFunc(c) {
			c.Next()
			return
		}

		// Get key for this request
		key := config.KeyFunc(c)

		// Increment counter
		count, expiresAt, err := config.Store.Increment(key, config.Window)
		if err != nil {
			// On error, allow the request but log it
			debugPrint("rate limiter error: %v", err)
			c.Next()
			return
		}

		// Set rate limit headers
		remaining := config.Max - count
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Max))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", expiresAt.Unix()))

		// Check if limit exceeded
		if count > config.Max {
			config.ErrorHandler(c)
			return
		}

		c.Next()
	}
}

// RateLimiterByUser returns a rate limiter that uses user ID as key
// Requires JWT middleware to be used before this middleware
func RateLimiterByUser(max int, window time.Duration) HandlerFunc {
	return RateLimiterWithConfig(RateLimiterConfig{
		Max:    max,
		Window: window,
		KeyFunc: func(c *Context) string {
			claims, exists := GetJWTClaims(c)
			if !exists || claims.UserID == "" {
				// Fallback to IP if no user ID
				return c.ClientIP()
			}
			return "user:" + claims.UserID
		},
	})
}

// RateLimiterByPath returns a rate limiter that uses request path as part of the key
func RateLimiterByPath(max int, window time.Duration) HandlerFunc {
	return RateLimiterWithConfig(RateLimiterConfig{
		Max:    max,
		Window: window,
		KeyFunc: func(c *Context) string {
			return c.ClientIP() + ":" + c.Request.URL.Path
		},
	})
}

// RateLimiterByAPIKey returns a rate limiter that uses API key as key
func RateLimiterByAPIKey(max int, window time.Duration, headerName string) HandlerFunc {
	if headerName == "" {
		headerName = "X-API-Key"
	}

	return RateLimiterWithConfig(RateLimiterConfig{
		Max:    max,
		Window: window,
		KeyFunc: func(c *Context) string {
			apiKey := c.Request.Header.Get(headerName)
			if apiKey == "" {
				// Fallback to IP if no API key
				return "ip:" + c.ClientIP()
			}
			return "apikey:" + apiKey
		},
	})
}

// BurstRateLimiter creates a rate limiter that allows bursts
// maxBurst: maximum burst size
// refillRate: tokens refilled per second
func BurstRateLimiter(maxBurst int, refillRate float64) HandlerFunc {
	store := &tokenBucketStore{
		buckets: make(map[string]*tokenBucket),
	}
	go store.cleanup()

	return func(c *Context) {
		key := c.ClientIP()
		allowed := store.allow(key, maxBurst, refillRate)

		if !allowed {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxBurst))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(429, H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// tokenBucketStore implements token bucket algorithm
type tokenBucketStore struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

func (s *tokenBucketStore) allow(key string, maxBurst int, refillRate float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	bucket, exists := s.buckets[key]

	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(maxBurst - 1),
			lastRefill: now,
		}
		s.buckets[key] = bucket
		return true
	}

	// Refill tokens
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * refillRate
	if bucket.tokens > float64(maxBurst) {
		bucket.tokens = float64(maxBurst)
	}
	bucket.lastRefill = now

	// Check if we have tokens
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

func (s *tokenBucketStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, bucket := range s.buckets {
			// Remove buckets that haven't been used in 10 minutes
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(s.buckets, key)
			}
		}
		s.mu.Unlock()
	}
}
