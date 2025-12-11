package main

import (
	"path/filepath"

	"github.com/yourusername/goTap"
)

func main() {
	r := goTap.Default()

	// Serve single file
	r.StaticFile("/favicon.ico", "./public/favicon.ico")

	// Serve static directory
	r.Static("/static", "./public")

	// Serve with custom FileSystem (no directory listing)
	publicPath, _ := filepath.Abs("./public")
	r.StaticFS("/assets", goTap.Dir(publicPath, false))

	// API routes
	r.GET("/api/status", func(c *goTap.Context) {
		c.JSON(200, goTap.H{
			"status":  "ok",
			"message": "Static file serving is working",
		})
	})

	// Root route
	r.GET("/", func(c *goTap.Context) {
		c.File("./public/index.html")
	})

	r.Run(":5066")
}
