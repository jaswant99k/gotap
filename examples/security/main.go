package main

import (
	"time"

	"github.com/jaswant99k/gotap"
)

func main() {
	r := goTap.Default()

	// JWT secret
	jwtSecret := "your-secret-key-change-this-in-production"

	// Add Transaction ID middleware globally
	r.Use(goTap.TransactionID())

	// Public routes
	r.POST("/login", func(c *goTap.Context) {
		// In production, verify credentials against database
		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "" || password == "" {
			c.JSON(400, goTap.H{
				"error": "Username and password required",
			})
			return
		}

		// Generate JWT token
		claims := goTap.JWTClaims{
			UserID:    "user123",
			Username:  username,
			Email:     username + "@example.com",
			Role:      "admin",
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			Issuer:    "goTap-security-demo",
		}

		token, err := goTap.GenerateJWT(jwtSecret, claims)
		if err != nil {
			c.JSON(500, goTap.H{
				"error": "Failed to generate token",
			})
			return
		}

		c.JSON(200, goTap.H{
			"token":          token,
			"transaction_id": goTap.GetTransactionID(c),
		})
	})

	// Protected routes with JWT
	authorized := r.Group("/api")
	authorized.Use(goTap.JWTAuth(jwtSecret))
	{
		authorized.GET("/profile", func(c *goTap.Context) {
			claims, _ := goTap.GetJWTClaims(c)
			c.JSON(200, goTap.H{
				"user_id":  claims.UserID,
				"username": claims.Username,
				"email":    claims.Email,
				"role":     claims.Role,
			})
		})

		// Admin-only route
		admin := authorized.Group("/admin")
		admin.Use(goTap.RequireRole("admin"))
		{
			admin.GET("/users", func(c *goTap.Context) {
				c.JSON(200, goTap.H{
					"users": []string{"user1", "user2", "user3"},
				})
			})
		}
	}

	// Rate-limited public API
	publicAPI := r.Group("/public")
	publicAPI.Use(goTap.RateLimiter(10, time.Minute)) // 10 requests per minute
	{
		publicAPI.GET("/status", func(c *goTap.Context) {
			c.JSON(200, goTap.H{
				"status":         "ok",
				"transaction_id": goTap.GetTransactionID(c),
			})
		})
	}

	// POS Terminal routes (IP whitelisted + JWT + Rate limited)
	allowedTerminalIPs := []string{
		"127.0.0.1",
		"::1",
		"192.168.1.0/24", // Local network
	}

	terminal := r.Group("/pos")
	terminal.Use(goTap.IPWhitelist(allowedTerminalIPs...))
	terminal.Use(goTap.JWTAuth(jwtSecret))
	terminal.Use(goTap.RateLimiter(100, time.Minute))
	{
		terminal.POST("/transaction", func(c *goTap.Context) {
			txID := goTap.GetTransactionID(c)
			claims, _ := goTap.GetJWTClaims(c)

			c.JSON(200, goTap.H{
				"transaction_id": txID,
				"terminal_user":  claims.Username,
				"status":         "processed",
				"message":        "Transaction recorded successfully",
			})
		})

		terminal.GET("/inventory/:sku", func(c *goTap.Context) {
			sku := c.Param("sku")
			c.JSON(200, goTap.H{
				"sku":      sku,
				"quantity": 42,
				"price":    19.99,
			})
		})
	}

	// Burst rate limiter example
	burst := r.Group("/burst")
	burst.Use(goTap.BurstRateLimiter(5, 1.0)) // 5 burst, 1 token/sec refill
	{
		burst.GET("/test", func(c *goTap.Context) {
			c.JSON(200, goTap.H{
				"message": "Burst test successful",
			})
		})
	}

	// Token refresh endpoint
	r.POST("/refresh", func(c *goTap.Context) {
		oldToken := c.Request.Header.Get("Authorization")
		if oldToken == "" {
			c.JSON(400, goTap.H{
				"error": "No token provided",
			})
			return
		}

		// Remove "Bearer " prefix
		if len(oldToken) > 7 {
			oldToken = oldToken[7:]
		}

		newToken, err := goTap.RefreshToken(oldToken, jwtSecret, 24*time.Hour)
		if err != nil {
			c.JSON(401, goTap.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(200, goTap.H{
			"token": newToken,
		})
	})

	r.Run(":5066")
}
