// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/yourusername/goTap"
)

func main() {
	// Create a goTap router with default middleware:
	// logger and recovery (crash-free) middleware
	r := goTap.Default()

	// Define a simple GET endpoint
	r.GET("/ping", func(c *goTap.Context) {
		c.JSON(200, goTap.H{
			"message": "pong",
		})
	})

	// Define a route with URL parameter
	r.GET("/hello/:name", func(c *goTap.Context) {
		name := c.Param("name")
		c.JSON(200, goTap.H{
			"message": "Hello " + name + "!",
		})
	})

	// Define a POST endpoint
	r.POST("/user", func(c *goTap.Context) {
		c.JSON(201, goTap.H{
			"status": "user created",
		})
	})

	// Group routes with common prefix
	api := r.Group("/api/v1")
	{
		api.GET("/status", func(c *goTap.Context) {
			c.JSON(200, goTap.H{
				"status":  "ok",
				"version": "1.0.0",
			})
		})

		api.GET("/users/:id", func(c *goTap.Context) {
			id := c.Param("id")
			c.JSON(200, goTap.H{
				"id":   id,
				"name": "User " + id,
			})
		})
	}

	// Run on port 8080 (default)
	// Server will listen on 0.0.0.0:8080
	r.Run() // listen and serve on 0.0.0.0:8080
}
