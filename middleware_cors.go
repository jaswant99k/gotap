package goTap

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CORSConfig defines the config for CORS middleware
type CORSConfig struct {
	// AllowOrigins is a list of origins that may access the resource
	// Default: []string{"*"}
	AllowOrigins []string

	// AllowMethods is a list of methods the client is allowed to use
	// Default: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	AllowMethods []string

	// AllowHeaders is a list of request headers that can be used when making the actual request
	// Default: []string{"Origin", "Content-Length", "Content-Type"}
	AllowHeaders []string

	// ExposeHeaders indicates which headers are safe to expose
	// Default: []string{}
	ExposeHeaders []string

	// AllowCredentials indicates whether the request can include user credentials
	// Default: false
	AllowCredentials bool

	// MaxAge indicates how long the results of a preflight request can be cached
	// Default: 12 hours
	MaxAge time.Duration

	// AllowWildcard allows wildcard subdomains (e.g., https://*.example.com)
	// Default: false
	AllowWildcard bool

	// AllowOriginFunc is a custom function to validate the origin
	// It takes the origin as an argument and returns true if allowed
	// This overrides AllowOrigins
	AllowOriginFunc func(origin string) bool
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
		AllowWildcard:    false,
	}
}

// CORS returns a middleware that adds CORS headers to responses
func CORS() HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig returns a CORS middleware with custom config
func CORSWithConfig(config CORSConfig) HandlerFunc {
	// Set defaults
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = []string{"*"}
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 12 * time.Hour
	}

	// Pre-compute values
	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	exposeHeaders := strings.Join(config.ExposeHeaders, ", ")
	maxAge := strconv.FormatInt(int64(config.MaxAge.Seconds()), 10)

	// Check if all origins are allowed
	allowAllOrigins := false
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
	}

	return func(c *Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowOrigin := ""

		if config.AllowOriginFunc != nil {
			// Use custom function if provided
			if config.AllowOriginFunc(origin) {
				allowOrigin = origin
			}
		} else if allowAllOrigins {
			// Allow all origins
			allowOrigin = "*"
		} else {
			// Check against whitelist
			for _, o := range config.AllowOrigins {
				if matchOrigin(origin, o, config.AllowWildcard) {
					allowOrigin = origin
					break
				}
			}
		}

		// Set CORS headers
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
		}

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowHeaders)

			if exposeHeaders != "" {
				c.Header("Access-Control-Expose-Headers", exposeHeaders)
			}

			c.Header("Access-Control-Max-Age", maxAge)

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// For actual requests, set expose headers
		if exposeHeaders != "" {
			c.Header("Access-Control-Expose-Headers", exposeHeaders)
		}

		c.Next()
	}
}

// matchOrigin checks if the origin matches the allowed pattern
func matchOrigin(origin, pattern string, allowWildcard bool) bool {
	if origin == pattern {
		return true
	}

	if !allowWildcard {
		return false
	}

	// Handle wildcard subdomains (e.g., https://*.example.com)
	if strings.Contains(pattern, "*") {
		// Convert pattern to a simple regex-like matching
		prefix := strings.Split(pattern, "*")[0]
		suffix := strings.Split(pattern, "*")[1]

		return strings.HasPrefix(origin, prefix) && strings.HasSuffix(origin, suffix)
	}

	return false
}
