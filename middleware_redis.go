package goTap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps redis.Client for middleware use
type RedisClient struct {
	Client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client wrapper
func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisClient{
		Client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// RedisCacheConfig holds configuration for Redis caching middleware
type RedisCacheConfig struct {
	// Redis client instance
	Client *RedisClient

	// TTL for cached responses (default: 5 minutes)
	TTL time.Duration

	// Key prefix for cache keys (default: "cache:")
	Prefix string

	// Skip cache for certain paths (e.g., ["/admin", "/api/auth"])
	SkipPaths []string

	// Only cache specific HTTP methods (default: ["GET"])
	CacheMethods []string

	// Custom key generator function (optional)
	KeyGenerator func(c *Context) string
}

// RedisCache returns a middleware that caches GET requests in Redis
func RedisCache(config RedisCacheConfig) HandlerFunc {
	// Set defaults
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}
	if config.Prefix == "" {
		config.Prefix = "cache:"
	}
	if len(config.CacheMethods) == 0 {
		config.CacheMethods = []string{"GET"}
	}
	if config.KeyGenerator == nil {
		config.KeyGenerator = defaultCacheKeyGenerator
	}

	return func(c *Context) {
		// Skip if client not provided
		if config.Client == nil || config.Client.Client == nil {
			c.Next()
			return
		}

		// Check if method is cacheable
		cacheable := false
		for _, method := range config.CacheMethods {
			if c.Request.Method == method {
				cacheable = true
				break
			}
		}
		if !cacheable {
			c.Next()
			return
		}

		// Check if path should be skipped
		for _, skipPath := range config.SkipPaths {
			if c.Request.URL.Path == skipPath {
				c.Next()
				return
			}
		}

		// Generate cache key
		cacheKey := config.Prefix + config.KeyGenerator(c)

		// Try to get from cache
		ctx := context.Background()
		cached, err := config.Client.Client.Get(ctx, cacheKey).Result()
		if err == nil {
			// Cache hit
			c.Header("X-Cache", "HIT")
			c.Header("X-Cache-Key", cacheKey)
			c.Data(200, "application/json", []byte(cached))
			c.Abort()
			return
		}

		// Cache miss - capture response
		c.Header("X-Cache", "MISS")
		c.Header("X-Cache-Key", cacheKey)

		// Create a custom writer to capture response
		writer := &cachedWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Store in cache if status is 200 and body exists
		if writer.status == 200 && len(writer.body) > 0 {
			config.Client.Client.Set(ctx, cacheKey, writer.body, config.TTL)
		}
	}
}

// cachedWriter captures response body for caching
type cachedWriter struct {
	ResponseWriter
	body   []byte
	status int
}

func (w *cachedWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return w.ResponseWriter.Write(data)
}

func (w *cachedWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// defaultCacheKeyGenerator generates a cache key from request
func defaultCacheKeyGenerator(c *Context) string {
	// Include method, path, and query string
	key := fmt.Sprintf("%s:%s", c.Request.Method, c.Request.URL.Path)
	if c.Request.URL.RawQuery != "" {
		key += "?" + c.Request.URL.RawQuery
	}

	// Hash the key for consistency
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// RedisInject injects Redis client into context for use in handlers
func RedisInject(client *RedisClient) HandlerFunc {
	return func(c *Context) {
		c.Set("redis", client)
		c.Next()
	}
}

// GetRedis retrieves Redis client from context
func GetRedis(c *Context) (*RedisClient, bool) {
	client, exists := c.Get("redis")
	if !exists {
		return nil, false
	}
	redisClient, ok := client.(*RedisClient)
	return redisClient, ok
}

// MustGetRedis retrieves Redis client from context or panics
func MustGetRedis(c *Context) *RedisClient {
	client, ok := GetRedis(c)
	if !ok {
		panic("Redis client not found in context")
	}
	return client
}

// RedisSessionConfig holds configuration for Redis session management
type RedisSessionConfig struct {
	// Redis client instance
	Client *RedisClient

	// Session TTL (default: 24 hours)
	TTL time.Duration

	// Cookie name (default: "session_id")
	CookieName string

	// Cookie path (default: "/")
	CookiePath string

	// Cookie domain (optional)
	CookieDomain string

	// Cookie secure flag (default: false)
	Secure bool

	// Cookie HttpOnly flag (default: true)
	HttpOnly bool
}

// RedisSession returns middleware for Redis-backed session management
func RedisSession(config RedisSessionConfig) HandlerFunc {
	// Set defaults
	if config.TTL == 0 {
		config.TTL = 24 * time.Hour
	}
	if config.CookieName == "" {
		config.CookieName = "session_id"
	}
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}

	return func(c *Context) {
		// Skip if client not provided
		if config.Client == nil || config.Client.Client == nil {
			c.Next()
			return
		}

		// Get session ID from cookie
		sessionID, err := c.Cookie(config.CookieName)
		if err != nil || sessionID == "" {
			// No session - create new one
			sessionID = generateSessionID()
			c.SetCookie(config.CookieName, sessionID, int(config.TTL.Seconds()),
				config.CookiePath, config.CookieDomain, config.Secure, config.HttpOnly)
		}

		// Load session data from Redis
		ctx := context.Background()
		sessionKey := "session:" + sessionID
		sessionData, _ := config.Client.Client.HGetAll(ctx, sessionKey).Result()

		// Create session object
		session := &Session{
			ID:     sessionID,
			Data:   sessionData,
			client: config.Client,
			key:    sessionKey,
			ttl:    config.TTL,
		}

		// Inject into context
		c.Set("session", session)

		// Process request
		c.Next()

		// Save session after request (if modified)
		if session.modified {
			session.Save()
		}

		// Refresh TTL
		config.Client.Client.Expire(ctx, sessionKey, config.TTL)
	}
}

// Session represents a user session stored in Redis
type Session struct {
	ID       string
	Data     map[string]string
	client   *RedisClient
	key      string
	ttl      time.Duration
	modified bool
}

// Get retrieves a value from session
func (s *Session) Get(key string) (string, bool) {
	val, exists := s.Data[key]
	return val, exists
}

// Set stores a value in session
func (s *Session) Set(key, value string) {
	if s.Data == nil {
		s.Data = make(map[string]string)
	}
	s.Data[key] = value
	s.modified = true
}

// Delete removes a value from session
func (s *Session) Delete(key string) {
	delete(s.Data, key)
	s.modified = true
}

// Save persists session data to Redis
func (s *Session) Save() error {
	if s.client == nil || s.client.Client == nil {
		return fmt.Errorf("redis client not available")
	}

	ctx := context.Background()
	pipe := s.client.Client.Pipeline()

	// Delete old data
	pipe.Del(ctx, s.key)

	// Set new data
	if len(s.Data) > 0 {
		pipe.HSet(ctx, s.key, s.Data)
		pipe.Expire(ctx, s.key, s.ttl)
	}

	_, err := pipe.Exec(ctx)
	s.modified = false
	return err
}

// Destroy removes session from Redis
func (s *Session) Destroy() error {
	if s.client == nil || s.client.Client == nil {
		return fmt.Errorf("redis client not available")
	}

	ctx := context.Background()
	return s.client.Client.Del(ctx, s.key).Err()
}

// GetSession retrieves session from context
func GetSession(c *Context) (*Session, bool) {
	session, exists := c.Get("session")
	if !exists {
		return nil, false
	}
	sess, ok := session.(*Session)
	return sess, ok
}

// MustGetSession retrieves session from context or panics
func MustGetSession(c *Context) *Session {
	session, ok := GetSession(c)
	if !ok {
		panic("Session not found in context")
	}
	return session
}

// generateSessionID creates a unique session ID
func generateSessionID() string {
	// Use timestamp + random nanotime for uniqueness
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("%d-%d", timestamp, timestamp/1000000)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes (32 hex chars)
}

// RedisHealthCheck returns middleware that checks Redis health
func RedisHealthCheck(client *RedisClient) HandlerFunc {
	return func(c *Context) {
		if client == nil || client.Client == nil {
			c.JSON(503, H{
				"status": "unhealthy",
				"redis":  "not configured",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		if err := client.Client.Ping(ctx).Err(); err != nil {
			c.JSON(503, H{
				"status": "unhealthy",
				"redis":  "connection failed",
				"error":  err.Error(),
			})
			c.Abort()
			return
		}

		c.JSON(200, H{
			"status": "healthy",
			"redis":  "connected",
		})
	}
}

// RedisPubSub provides pub/sub functionality for real-time updates
type RedisPubSub struct {
	client *RedisClient
	pubsub *redis.PubSub
}

// NewRedisPubSub creates a new pub/sub instance
func NewRedisPubSub(client *RedisClient, channels ...string) *RedisPubSub {
	ctx := context.Background()
	pubsub := client.Client.Subscribe(ctx, channels...)

	return &RedisPubSub{
		client: client,
		pubsub: pubsub,
	}
}

// Publish sends a message to a channel
func (ps *RedisPubSub) Publish(channel, message string) error {
	ctx := context.Background()
	return ps.client.Client.Publish(ctx, channel, message).Err()
}

// Receive returns the channel for receiving messages
func (ps *RedisPubSub) Receive() <-chan *redis.Message {
	return ps.pubsub.Channel()
}

// Close closes the pub/sub connection
func (ps *RedisPubSub) Close() error {
	return ps.pubsub.Close()
}
