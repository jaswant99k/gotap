package main

import (
	"log"

	goTap "github.com/jaswant99k/gotap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model using goTap.Model
type User struct {
	goTap.Model
	Username     string       `gorm:"uniqueIndex;not null" json:"username" example:"john_doe"`
	Email        string       `gorm:"uniqueIndex;not null" json:"email" example:"john@example.com"`
	PasswordHash string       `gorm:"not null" json:"-"`
	Role         string       `gorm:"default:'user'" json:"role" example:"user"`
	IsActive     bool         `gorm:"default:true" json:"is_active" example:"true"`
	Permissions  []Permission `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
}

// Permission model using goTap.Model
type Permission struct {
	goTap.Model
	Name        string `gorm:"uniqueIndex;not null" json:"name" example:"read:users"`
	Description string `json:"description" example:"Can read user data"`
}

// Product model using goTap.Model
type Product struct {
	goTap.Model
	Name        string  `gorm:"not null" json:"name" example:"Laptop"`
	SKU         string  `gorm:"uniqueIndex;not null" json:"sku" example:"LAP-001"`
	Description string  `json:"description" example:"High-performance laptop"`
	Price       float64 `gorm:"not null" json:"price" example:"999.99"`
	Stock       int     `gorm:"default:0" json:"stock" example:"10"`
	Category    string  `gorm:"index" json:"category" example:"Electronics"`
	IsActive    bool    `gorm:"default:true" json:"is_active" example:"true"`
}

// Order model demonstrating relationships
type Order struct {
	goTap.Model
	UserID      uint        `gorm:"not null;index" json:"user_id" example:"1"`
	User        User        `gorm:"foreignKey:UserID" json:"user"`
	OrderItems  []OrderItem `gorm:"foreignKey:OrderID" json:"order_items"`
	TotalAmount float64     `gorm:"not null" json:"total_amount" example:"1299.99"`
	Status      string      `gorm:"default:'pending'" json:"status" example:"pending"`
}

// OrderItem model for many-to-many with additional fields
type OrderItem struct {
	goTap.Model
	OrderID   uint    `gorm:"not null;index" json:"order_id" example:"1"`
	ProductID uint    `gorm:"not null;index" json:"product_id" example:"1"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `gorm:"not null" json:"quantity" example:"2"`
	Price     float64 `gorm:"not null" json:"price" example:"999.99"`
}

func main() {
	// Initialize goTap
	r := goTap.Default()

	// Setup database
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}

	// Auto migrate models
	err = db.AutoMigrate(&User{}, &Permission{}, &Product{}, &Order{}, &OrderItem{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Inject GORM instance as middleware
	r.Use(goTap.GormInject(db))

	// Routes
	r.GET("/health", func(c *goTap.Context) {
		c.JSON(200, goTap.H{"status": "ok"})
	})

	// User routes
	r.POST("/users", createUser)
	r.GET("/users/:id", getUser)
	r.GET("/users", listUsers)
	r.PUT("/users/:id", updateUser)
	r.DELETE("/users/:id", deleteUser)

	// Product routes
	r.POST("/products", createProduct)
	r.GET("/products/:id", getProduct)
	r.GET("/products", listProducts)

	// Order routes
	r.POST("/orders", createOrder)
	r.GET("/orders/:id", getOrder)

	// Run server
	log.Println("Server starting on :8080...")
	r.Run(":8080")
}

// User handlers
func createUser(c *goTap.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	db := goTap.MustGetGorm(c)
	if err := db.Create(&user).Error; err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(201, user)
}

func getUser(c *goTap.Context) {
	id := c.Param("id")
	var user User

	db := goTap.MustGetGorm(c)
	// Preload permissions relationship
	if err := db.Preload("Permissions").First(&user, id).Error; err != nil {
		c.JSON(404, goTap.H{"error": "User not found"})
		return
	}

	c.JSON(200, user)
}

func listUsers(c *goTap.Context) {
	var users []User

	db := goTap.MustGetGorm(c)
	if err := db.Preload("Permissions").Find(&users).Error; err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, users)
}

func updateUser(c *goTap.Context) {
	id := c.Param("id")
	var user User

	db := goTap.MustGetGorm(c)
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(404, goTap.H{"error": "User not found"})
		return
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, user)
}

func deleteUser(c *goTap.Context) {
	id := c.Param("id")
	var user User

	db := goTap.MustGetGorm(c)
	// Soft delete (sets DeletedAt timestamp)
	if err := db.Delete(&user, id).Error; err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, goTap.H{"message": "User deleted successfully"})
}

// Product handlers
func createProduct(c *goTap.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	db := goTap.MustGetGorm(c)
	if err := db.Create(&product).Error; err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(201, product)
}

func getProduct(c *goTap.Context) {
	id := c.Param("id")
	var product Product

	db := goTap.MustGetGorm(c)
	if err := db.First(&product, id).Error; err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	c.JSON(200, product)
}

func listProducts(c *goTap.Context) {
	var products []Product

	db := goTap.MustGetGorm(c)
	// Support filtering by category
	query := db
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Find(&products).Error; err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, products)
}

// Order handlers
func createOrder(c *goTap.Context) {
	var order Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	db := goTap.MustGetGorm(c)

	// Start transaction
	tx := db.Begin()

	// Create order
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	// Update product stock (example)
	for _, item := range order.OrderItems {
		if err := tx.Model(&Product{}).Where("id = ?", item.ProductID).
			Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
			tx.Rollback()
			c.JSON(500, goTap.H{"error": "Failed to update stock"})
			return
		}
	}

	tx.Commit()
	c.JSON(201, order)
}

func getOrder(c *goTap.Context) {
	id := c.Param("id")
	var order Order

	db := goTap.MustGetGorm(c)
	// Preload all relationships
	if err := db.Preload("User").Preload("OrderItems.Product").First(&order, id).Error; err != nil {
		c.JSON(404, goTap.H{"error": "Order not found"})
		return
	}

	c.JSON(200, order)
}
