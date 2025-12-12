package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaswant99k/gotap"
)

func main() {
	router := goTap.Default()

	// Simulate long-running request (e.g., processing payment)
	router.GET("/process", func(c *goTap.Context) {
		duration := c.DefaultQuery("duration", "5")
		sleepTime, _ := time.ParseDuration(duration + "s")

		log.Printf("Processing request for %v...", sleepTime)

		// Simulate long processing
		time.Sleep(sleepTime)

		c.JSON(http.StatusOK, goTap.H{
			"message":  "Processing complete",
			"duration": sleepTime.String(),
		})
	})

	// Quick response endpoint
	router.GET("/ping", func(c *goTap.Context) {
		c.JSON(http.StatusOK, goTap.H{
			"message": "pong",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Health check endpoint
	router.GET("/health", func(c *goTap.Context) {
		c.JSON(http.StatusOK, goTap.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Start server with graceful shutdown support
	srv := router.RunServer(":5066")

	// Wait for interrupt signal to gracefully shutdown the server
	// with a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	// Kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	// The context is used to inform the server it has 10 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
