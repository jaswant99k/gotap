package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jaswant99k/gotap"
)

type Product struct {
	ID          string  `json:"id" xml:"id"`
	Name        string  `json:"name" xml:"name"`
	Price       float64 `json:"price" xml:"price"`
	Description string  `json:"description" xml:"description"`
	Category    string  `json:"category" xml:"category"`
}

func main() {
	app := goTap.Default()

	// Load HTML templates
	app.LoadHTMLGlob("templates/*")

	// ========== JSON Rendering ==========
	app.GET("/api/products", func(c *goTap.Context) {
		products := []Product{
			{ID: "P001", Name: "Laptop", Price: 999.99, Description: "High-performance laptop", Category: "Electronics"},
			{ID: "P002", Name: "Mouse", Price: 29.99, Description: "Wireless mouse", Category: "Electronics"},
			{ID: "P003", Name: "Keyboard", Price: 79.99, Description: "Mechanical keyboard", Category: "Electronics"},
		}

		c.JSON(http.StatusOK, goTap.H{
			"products": products,
			"total":    len(products),
		})
	})

	// ========== XML Rendering ==========
	app.GET("/api/products.xml", func(c *goTap.Context) {
		products := []Product{
			{ID: "P001", Name: "Laptop", Price: 999.99, Description: "High-performance laptop", Category: "Electronics"},
		}

		c.XML(http.StatusOK, goTap.H{
			"products": products,
		})
	})

	// ========== YAML Rendering ==========
	app.GET("/api/products.yaml", func(c *goTap.Context) {
		c.YAML(http.StatusOK, goTap.H{
			"name":     "Product Catalog",
			"version":  "1.0",
			"products": "3",
		})
	})

	// ========== HTML Rendering ==========
	app.GET("/", func(c *goTap.Context) {
		c.HTML(http.StatusOK, "index.html", goTap.H{
			"title":   "goTap Store",
			"message": "Welcome to our online store!",
			"year":    time.Now().Year(),
		})
	})

	app.GET("/products", func(c *goTap.Context) {
		products := []Product{
			{ID: "P001", Name: "Laptop", Price: 999.99, Description: "High-performance laptop", Category: "Electronics"},
			{ID: "P002", Name: "Mouse", Price: 29.99, Description: "Wireless mouse", Category: "Electronics"},
			{ID: "P003", Name: "Keyboard", Price: 79.99, Description: "Mechanical keyboard", Category: "Electronics"},
		}

		c.HTML(http.StatusOK, "products.html", goTap.H{
			"title":    "Products",
			"products": products,
		})
	})

	app.GET("/product/:id", func(c *goTap.Context) {
		id := c.Param("id")

		product := Product{
			ID:          id,
			Name:        "Sample Product",
			Price:       99.99,
			Description: "This is a sample product",
			Category:    "Electronics",
		}

		c.HTML(http.StatusOK, "product_detail.html", goTap.H{
			"title":   product.Name,
			"product": product,
		})
	})

	// ========== Content Negotiation ==========
	app.GET("/api/data", func(c *goTap.Context) {
		data := Product{
			ID:          "P001",
			Name:        "Laptop",
			Price:       999.99,
			Description: "High-performance laptop",
			Category:    "Electronics",
		}

		c.Negotiate(http.StatusOK, goTap.Negotiate{
			Offered:  []string{"application/json", "application/xml", "text/html"},
			JSONData: data,
			XMLData:  data,
			HTMLName: "product_detail.html",
			HTMLData: goTap.H{"title": "Product", "product": data},
		})
	})

	// ========== Server-Sent Events (SSE) ==========
	app.GET("/events", func(c *goTap.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// Send initial event
		c.SSE("connected", goTap.H{"message": "Connected to event stream"})
	})

	app.GET("/events/stream", func(c *goTap.Context) {
		c.Stream(func(w http.ResponseWriter) bool {
			// Send event every second
			c.SSE("update", goTap.H{
				"time":    time.Now().Format(time.RFC3339),
				"message": "Periodic update",
			})
			time.Sleep(1 * time.Second)
			return true // Keep connection alive
		})
	})

	// ========== Cookies ==========
	app.GET("/cookie/set", func(c *goTap.Context) {
		c.SetCookie("user_id", "12345", 3600, "/", "", false, true)
		c.SetCookie("session", "abc123", 7200, "/", "", false, true)

		c.JSON(http.StatusOK, goTap.H{
			"message": "Cookies set successfully",
		})
	})

	app.GET("/cookie/get", func(c *goTap.Context) {
		userID, err1 := c.Cookie("user_id")
		session, err2 := c.Cookie("session")

		c.JSON(http.StatusOK, goTap.H{
			"user_id":       userID,
			"user_id_error": err1,
			"session":       session,
			"session_error": err2,
		})
	})

	// ========== Redirect ==========
	app.GET("/redirect", func(c *goTap.Context) {
		c.Redirect(http.StatusMovedPermanently, "/")
	})

	app.GET("/redirect-temp", func(c *goTap.Context) {
		c.Redirect(http.StatusFound, "/products")
	})

	// ========== Raw Data ==========
	app.GET("/api/raw", func(c *goTap.Context) {
		data := []byte("This is raw binary data")
		c.Data(http.StatusOK, "application/octet-stream", data)
	})

	// ========== String Response ==========
	app.GET("/api/text", func(c *goTap.Context) {
		c.String(http.StatusOK, "Plain text response: %s", "Hello World")
	})

	// ========== File Download ==========
	app.GET("/download", func(c *goTap.Context) {
		c.FileAttachment("./main.go", "goTap-example.go")
	})

	// Start server
	fmt.Println("🎨 goTap Rendering Example Server")
	fmt.Println("📍 Server running at http://localhost:5066")
	fmt.Println("\n📚 Available Endpoints:")
	fmt.Println("  GET    /                      - HTML homepage")
	fmt.Println("  GET    /products              - HTML product list")
	fmt.Println("  GET    /product/:id           - HTML product detail")
	fmt.Println("  GET    /api/products          - JSON product list")
	fmt.Println("  GET    /api/products.xml      - XML product list")
	fmt.Println("  GET    /api/products.yaml     - YAML config")
	fmt.Println("  GET    /api/data              - Content negotiation (JSON/XML/HTML)")
	fmt.Println("  GET    /events                - Server-Sent Events (single)")
	fmt.Println("  GET    /events/stream         - Server-Sent Events (stream)")
	fmt.Println("  GET    /cookie/set            - Set cookies")
	fmt.Println("  GET    /cookie/get            - Get cookies")
	fmt.Println("  GET    /redirect              - Permanent redirect")
	fmt.Println("  GET    /redirect-temp         - Temporary redirect")
	fmt.Println("  GET    /api/raw               - Raw binary data")
	fmt.Println("  GET    /api/text              - Plain text")
	fmt.Println("  GET    /download              - File download")
	fmt.Println()

	app.Run(":5066")
}
