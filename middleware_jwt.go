// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// JWT errors
var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrExpiredToken      = errors.New("token has expired")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrMissingToken      = errors.New("missing authorization token")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID    string                 `json:"user_id,omitempty"`
	Username  string                 `json:"username,omitempty"`
	Email     string                 `json:"email,omitempty"`
	Role      string                 `json:"role,omitempty"`
	ExpiresAt int64                  `json:"exp"`
	IssuedAt  int64                  `json:"iat"`
	Issuer    string                 `json:"iss,omitempty"`
	Subject   string                 `json:"sub,omitempty"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// JWTConfig holds JWT middleware configuration
type JWTConfig struct {
	// Secret key for signing tokens
	Secret string

	// TokenLookup is a string in the form of "<source>:<name>" that is used
	// to extract token from the request.
	// Optional. Default value "header:Authorization".
	// Possible values:
	// - "header:<name>"
	// - "query:<name>"
	// - "cookie:<name>"
	TokenLookup string

	// TokenHeadName is a string in the header. Default value is "Bearer"
	TokenHeadName string

	// TimeFunc provides the current time. You can override it for testing.
	TimeFunc func() time.Time

	// ErrorHandler defines a function which is executed when an error occurs.
	ErrorHandler func(*Context, error)

	// SuccessHandler defines a function which is executed after successful token validation.
	SuccessHandler func(*Context, *JWTClaims)
}

// JWTAuth returns a JWT authentication middleware
func JWTAuth(secret string) HandlerFunc {
	return JWTAuthWithConfig(JWTConfig{
		Secret: secret,
	})
}

// JWTAuthWithConfig returns a JWT authentication middleware with config
func JWTAuthWithConfig(config JWTConfig) HandlerFunc {
	if config.Secret == "" {
		panic("JWT secret cannot be empty")
	}

	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}

	if config.TokenHeadName == "" {
		config.TokenHeadName = "Bearer"
	}

	if config.TimeFunc == nil {
		config.TimeFunc = time.Now
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = func(c *Context, err error) {
			c.JSON(401, H{
				"error":   "Unauthorized",
				"message": err.Error(),
			})
			c.Abort()
		}
	}

	// Parse TokenLookup
	parts := strings.Split(config.TokenLookup, ":")
	if len(parts) != 2 {
		panic("invalid TokenLookup format")
	}
	extractor := parts[0]
	extractorKey := parts[1]

	return func(c *Context) {
		var token string

		// Extract token based on TokenLookup
		switch extractor {
		case "header":
			auth := c.Request.Header.Get(extractorKey)
			if auth == "" {
				config.ErrorHandler(c, ErrMissingToken)
				return
			}

			// Check for Bearer prefix
			if len(auth) > len(config.TokenHeadName)+1 {
				if strings.EqualFold(auth[:len(config.TokenHeadName)], config.TokenHeadName) {
					token = auth[len(config.TokenHeadName)+1:]
				}
			}

			if token == "" {
				config.ErrorHandler(c, ErrInvalidAuthHeader)
				return
			}

		case "query":
			token = c.Query(extractorKey)
			if token == "" {
				config.ErrorHandler(c, ErrMissingToken)
				return
			}

		case "cookie":
			cookie, err := c.Request.Cookie(extractorKey)
			if err != nil {
				config.ErrorHandler(c, ErrMissingToken)
				return
			}
			token = cookie.Value
		}

		// Parse and validate token
		claims, err := parseJWT(token, config.Secret, config.TimeFunc)
		if err != nil {
			config.ErrorHandler(c, err)
			return
		}

		// Store claims in context
		c.Set("jwt_claims", claims)
		c.Set("user_id", claims.UserID)

		// Call success handler if provided
		if config.SuccessHandler != nil {
			config.SuccessHandler(c, claims)
		}

		c.Next()
	}
}

// GenerateJWT generates a new JWT token with the given claims
func GenerateJWT(secret string, claims JWTClaims) (string, error) {
	if claims.IssuedAt == 0 {
		claims.IssuedAt = time.Now().Unix()
	}

	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Create payload
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Create signature
	message := headerB64 + "." + payloadB64
	signature := createSignature(message, secret)

	// Return token
	return message + "." + signature, nil
}

// parseJWT parses and validates a JWT token
func parseJWT(tokenString, secret string, timeFunc func() time.Time) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Verify signature
	message := parts[0] + "." + parts[1]
	expectedSignature := createSignature(message, secret)
	if parts[2] != expectedSignature {
		return nil, ErrInvalidSignature
	}

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Parse claims
	var claims JWTClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	// Check expiration
	if claims.ExpiresAt > 0 && timeFunc().Unix() > claims.ExpiresAt {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

// createSignature creates HMAC-SHA256 signature
func createSignature(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// GetJWTClaims retrieves JWT claims from context
func GetJWTClaims(c *Context) (*JWTClaims, bool) {
	claims, exists := c.Get("jwt_claims")
	if !exists {
		return nil, false
	}
	jwtClaims, ok := claims.(*JWTClaims)
	return jwtClaims, ok
}

// RefreshToken generates a new token with extended expiration
func RefreshToken(oldToken, secret string, extendDuration time.Duration) (string, error) {
	claims, err := parseJWT(oldToken, secret, time.Now)
	if err != nil {
		// Allow expired tokens to be refreshed
		if !errors.Is(err, ErrExpiredToken) {
			return "", err
		}
		// Parse without time validation
		parts := strings.Split(oldToken, ".")
		if len(parts) != 3 {
			return "", ErrInvalidToken
		}
		payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return "", ErrInvalidToken
		}
		if err := json.Unmarshal(payloadJSON, &claims); err != nil {
			return "", ErrInvalidToken
		}
	}

	// Extend expiration
	claims.ExpiresAt = time.Now().Add(extendDuration).Unix()
	claims.IssuedAt = time.Now().Unix()

	return GenerateJWT(secret, *claims)
}

// RequireRole returns a middleware that checks if the user has the required role
func RequireRole(requiredRole string) HandlerFunc {
	return func(c *Context) {
		claims, exists := GetJWTClaims(c)
		if !exists {
			c.JSON(401, H{
				"error":   "Unauthorized",
				"message": "JWT claims not found",
			})
			c.Abort()
			return
		}

		if claims.Role != requiredRole {
			c.JSON(403, H{
				"error":   "Forbidden",
				"message": fmt.Sprintf("Required role: %s", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole returns a middleware that checks if the user has any of the required roles
func RequireAnyRole(roles ...string) HandlerFunc {
	return func(c *Context) {
		claims, exists := GetJWTClaims(c)
		if !exists {
			c.JSON(401, H{
				"error":   "Unauthorized",
				"message": "JWT claims not found",
			})
			c.Abort()
			return
		}

		for _, role := range roles {
			if claims.Role == role {
				c.Next()
				return
			}
		}

		c.JSON(403, H{
			"error":   "Forbidden",
			"message": "Insufficient permissions",
		})
		c.Abort()
	}
}
