package main

import (
	"fmt"
	"net/http"

	"github.com/jaswant99k/gotap"
)

// Transaction represents a POS transaction
type Transaction struct {
	ID          string  `json:"id" uri:"id"`
	Amount      float64 `json:"amount" form:"amount" validate:"required,min=0.01"`
	Currency    string  `json:"currency" form:"currency" validate:"required,oneof=USD EUR GBP"`
	Description string  `json:"description" form:"description" validate:"max=500"`
	CustomerID  string  `json:"customer_id" form:"customer_id"`
	Items       []Item  `json:"items" validate:"required,min=1"`
}

// Item represents a line item in a transaction
type Item struct {
	ProductID string  `json:"product_id" validate:"required"`
	Name      string  `json:"name" validate:"required,min=1,max=200"`
	Quantity  int     `json:"quantity" validate:"required,min=1"`
	Price     float64 `json:"price" validate:"required,min=0"`
}

// UserQuery represents query parameters for user search
type UserQuery struct {
	Page     int    `form:"page" validate:"min=1"`
	PageSize int    `form:"page_size" validate:"min=1,max=100"`
	Search   string `form:"search"`
	Status   string `form:"status" validate:"oneof=active inactive all"`
}

// AuthHeader represents authentication headers
type AuthHeader struct {
	Authorization string `header:"Authorization" validate:"required"`
	APIKey        string `header:"X-API-Key"`
	DeviceID      string `header:"X-Device-ID"`
}

// ProductForm represents a product creation form
type ProductForm struct {
	Name        string   `form:"name" validate:"required,min=3,max=100"`
	Price       float64  `form:"price" validate:"required,min=0.01"`
	Category    string   `form:"category" validate:"required"`
	Description string   `form:"description" validate:"max=1000"`
	Tags        []string `form:"tags"`
	InStock     bool     `form:"in_stock"`
}

// CustomerRegistration represents customer registration with validation
type CustomerRegistration struct {
	FirstName   string `json:"first_name" validate:"required,min=2,max=50"`
	LastName    string `json:"last_name" validate:"required,min=2,max=50"`
	Email       string `json:"email" validate:"required,email"`
	Phone       string `json:"phone" validate:"numeric,min=10,max=15"`
	Website     string `json:"website" validate:"url"`
	Age         int    `json:"age" validate:"min=18,max=120"`
	AcceptTerms bool   `json:"accept_terms" validate:"required"`
}

func main() {
	app := goTap.Default()

	// ========== JSON Binding Examples ==========

	// Create transaction (JSON binding with validation)
	app.POST("/api/transactions", func(c *goTap.Context) {
		var txn Transaction

		// Bind JSON and validate automatically
		if err := c.ShouldBindJSON(&txn); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{
				"error":   "Invalid request data",
				"details": err.Error(),
			})
			return
		}

		// Calculate total
		var total float64
		for _, item := range txn.Items {
			total += item.Price * float64(item.Quantity)
		}

		c.JSON(http.StatusCreated, goTap.H{
			"message":     "Transaction created successfully",
			"transaction": txn,
			"total":       total,
			"currency":    txn.Currency,
		})
	})

	// ========== Query Parameter Binding ==========

	// Search users with query parameters
	app.GET("/api/users", func(c *goTap.Context) {
		var query UserQuery

		// Set defaults
		query.Page = 1
		query.PageSize = 20
		query.Status = "all"

		// Bind query parameters
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, goTap.H{
			"page":      query.Page,
			"page_size": query.PageSize,
			"search":    query.Search,
			"status":    query.Status,
			"results":   []string{"user1", "user2", "user3"}, // Mock data
			"total":     3,
		})
	})

	// ========== URI Parameter Binding ==========

	// Get transaction by ID
	app.GET("/api/transactions/:id", func(c *goTap.Context) {
		var uri struct {
			ID string `uri:"id" validate:"required"`
		}

		if err := c.ShouldBindUri(&uri); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, goTap.H{
			"transaction_id": uri.ID,
			"amount":         299.99,
			"currency":       "USD",
			"status":         "completed",
		})
	})

	// ========== Header Binding ==========

	// Protected endpoint with header authentication
	app.GET("/api/protected", func(c *goTap.Context) {
		var headers AuthHeader

		if err := c.ShouldBindHeader(&headers); err != nil {
			c.JSON(http.StatusUnauthorized, goTap.H{"error": "Missing authentication headers"})
			return
		}

		c.JSON(http.StatusOK, goTap.H{
			"message":   "Access granted",
			"device_id": headers.DeviceID,
			"api_key":   headers.APIKey,
		})
	})

	// ========== Form Binding ==========

	// Create product from HTML form
	app.POST("/api/products", func(c *goTap.Context) {
		var form ProductForm

		// Automatically binds based on Content-Type
		if err := c.ShouldBind(&form); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, goTap.H{
			"message": "Product created successfully",
			"product": form,
		})
	})

	// ========== XML Binding ==========

	// Accept XML data
	app.POST("/api/transactions/xml", func(c *goTap.Context) {
		var txn Transaction

		if err := c.ShouldBindXML(&txn); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, goTap.H{
			"message":     "Transaction created from XML",
			"transaction": txn,
		})
	})

	// ========== Multipart Form with File Upload ==========

	// Upload product image
	app.POST("/api/products/:id/image", func(c *goTap.Context) {
		// Get product ID from URI
		productID := c.Param("id")

		// Get uploaded file
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": "No file uploaded"})
			return
		}

		// Save file (in production, validate file type, size, etc.)
		filename := fmt.Sprintf("./uploads/%s_%s", productID, file.Filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusInternalServerError, goTap.H{"error": "Failed to save file"})
			return
		}

		c.JSON(http.StatusOK, goTap.H{
			"message":    "Image uploaded successfully",
			"product_id": productID,
			"filename":   file.Filename,
			"size":       file.Size,
			"path":       filename,
		})
	})

	// ========== Advanced Validation Example ==========

	// Customer registration with comprehensive validation
	app.POST("/api/customers/register", func(c *goTap.Context) {
		var customer CustomerRegistration

		if err := c.ShouldBindJSON(&customer); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{
				"error":   "Validation failed",
				"details": err.Error(),
			})
			return
		}

		if !customer.AcceptTerms {
			c.JSON(http.StatusBadRequest, goTap.H{
				"error": "You must accept the terms and conditions",
			})
			return
		}

		c.JSON(http.StatusCreated, goTap.H{
			"message":  "Customer registered successfully",
			"customer": customer,
		})
	})

	// ========== Mixed Binding Example ==========

	// Update transaction with multiple binding sources
	app.PUT("/api/transactions/:id", func(c *goTap.Context) {
		// Bind URI parameter
		var uri struct {
			ID string `uri:"id" validate:"required"`
		}
		if err := c.ShouldBindUri(&uri); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": "Invalid transaction ID"})
			return
		}

		// Bind headers for auth
		var headers AuthHeader
		if err := c.ShouldBindHeader(&headers); err != nil {
			c.JSON(http.StatusUnauthorized, goTap.H{"error": "Missing authentication"})
			return
		}

		// Bind JSON body
		var update struct {
			Status      string `json:"status" validate:"required,oneof=pending completed cancelled"`
			Description string `json:"description" validate:"max=500"`
		}
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, goTap.H{
			"message":        "Transaction updated",
			"transaction_id": uri.ID,
			"new_status":     update.Status,
			"description":    update.Description,
		})
	})

	// ========== Reusable Body Binding ==========

	// Endpoint that needs to read body multiple times
	app.POST("/api/validate-and-store", func(c *goTap.Context) {
		var txn1 Transaction

		// First validation
		if err := c.ShouldBindBodyWith(&txn1, goTap.JSON); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": err.Error()})
			return
		}

		// Can read body again (cached)
		var txn2 Transaction
		if err := c.ShouldBindBodyWith(&txn2, goTap.JSON); err != nil {
			c.JSON(http.StatusBadRequest, goTap.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, goTap.H{
			"message": "Body read twice successfully",
			"data":    txn1,
		})
	})

	// Start server
	fmt.Println("🚀 goTap Binding Example Server")
	fmt.Println("📍 Server running at http://localhost:5066")
	fmt.Println("\n📚 Available Endpoints:")
	fmt.Println("  POST   /api/transactions          - Create transaction (JSON)")
	fmt.Println("  GET    /api/users                 - Search users (Query params)")
	fmt.Println("  GET    /api/transactions/:id      - Get transaction (URI param)")
	fmt.Println("  GET    /api/protected             - Protected route (Headers)")
	fmt.Println("  POST   /api/products              - Create product (Form)")
	fmt.Println("  POST   /api/transactions/xml      - Create transaction (XML)")
	fmt.Println("  POST   /api/products/:id/image    - Upload image (Multipart)")
	fmt.Println("  POST   /api/customers/register    - Register customer (Validation)")
	fmt.Println("  PUT    /api/transactions/:id      - Update transaction (Mixed)")
	fmt.Println("  POST   /api/validate-and-store    - Reusable body binding")
	fmt.Println()

	app.Run(":5066")
}
