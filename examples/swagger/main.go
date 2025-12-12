package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jaswant99k/gotap"
	"github.com/jaswant99k/gotap/examples/swagger/docs"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// @title           goTap Swagger Example API
// @version         1.0
// @description     Example API demonstrating Swagger/OpenAPI documentation with goTap framework
// @termsOfService  http://swagger.io/terms/

// @contact.name   goTap API Support
// @contact.url    https://github.com/jaswant99k/gotap
// @contact.email  support@gotap.dev

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

var (
	jwtSecret = "my-super-secret-jwt-key-change-in-production-min-32-chars"
	db        *gorm.DB
)

// User represents a user account
type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;not null" json:"username" example:"john_doe"`
	Email        string `gorm:"uniqueIndex;not null" json:"email" example:"john@example.com"`
	PasswordHash string `gorm:"not null" json:"-"`
	Role         string `gorm:"default:'user'" json:"role" example:"user" enums:"admin,user"`
	IsActive     bool   `gorm:"default:true" json:"is_active" example:"true"`
} // @name User

// Product represents a product
type Product struct {
	gorm.Model
	Name        string  `gorm:"not null" json:"name" example:"Laptop"`
	SKU         string  `gorm:"uniqueIndex;not null" json:"sku" example:"LAP001"`
	Description string  `json:"description" example:"High-performance laptop"`
	Price       float64 `gorm:"not null" json:"price" example:"999.99"`
	Stock       int     `gorm:"default:0" json:"stock" example:"10"`
	Category    string  `json:"category" example:"Electronics"`
} // @name Product

// RegisterRequest represents user registration data
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3" example:"john_doe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
} // @name RegisterRequest

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"admin@example.com"`
	Password string `json:"password" binding:"required" example:"admin123"`
} // @name LoginRequest

// CreateProductRequest represents product creation data
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required" example:"Laptop"`
	SKU         string  `json:"sku" binding:"required" example:"LAP001"`
	Description string  `json:"description" example:"High-performance laptop"`
	Price       float64 `json:"price" binding:"required,gt=0" example:"999.99"`
	Stock       int     `json:"stock" binding:"gte=0" example:"10"`
	Category    string  `json:"category" example:"Electronics"`
} // @name CreateProductRequest

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
} // @name ErrorResponse

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message" example:"Success"`
	Data    interface{} `json:"data,omitempty"`
} // @name SuccessResponse

func main() {
	// Initialize database
	var err error
	db, err = gorm.Open(sqlite.Open("swagger_example.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}

	// Auto-migrate
	db.AutoMigrate(&User{}, &Product{})

	// Seed default data
	seedData()

	// Initialize goTap
	r := goTap.Default()

	// Determine port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Update Swagger host dynamically to match the running port
	docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(addr)

	// Setup Swagger UI (public access)
	goTap.SetupSwagger(r, "/swagger")

	// Health check
	r.GET("/health", healthCheck)

	// API routes
	api := r.Group("/api")
	{
		// Public routes
		api.POST("/auth/register", register)
		api.POST("/auth/login", login)

		// Protected routes
		auth := api.Group("")
		auth.Use(goTap.JWTAuth(jwtSecret))
		{
			auth.GET("/auth/profile", getProfile)
			auth.GET("/products", getProducts)
			auth.GET("/products/:id", getProduct)
		}

		// Admin routes
		admin := api.Group("/admin")
		admin.Use(goTap.JWTAuth(jwtSecret))
		admin.Use(goTap.RequireRole("admin"))
		{
			admin.POST("/products", createProduct)
			admin.PUT("/products/:id", updateProduct)
			admin.DELETE("/products/:id", deleteProduct)
			admin.GET("/users", listUsers)
		}
	}

	fmt.Println("🚀 Server starting on http://localhost:" + port)
	fmt.Println("📚 Swagger UI: http://localhost:" + port + "/swagger/index.html")
	fmt.Println("🔑 Default admin: admin@example.com / admin123")
	fmt.Println()

	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// healthCheck godoc
// @Summary      Health check
// @Description  Check if the API is running
// @Tags         System
// @Produce      json
// @Success      200  {object}  SuccessResponse
// @Router       /health [get]
func healthCheck(c *goTap.Context) {
	c.JSON(200, goTap.H{
		"status": "ok",
		"time":   time.Now(),
	})
}

// register godoc
// @Summary      Register new user
// @Description  Create a new user account
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration data"
// @Success      201  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Router       /auth/register [post]
func register(c *goTap.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		c.JSON(500, goTap.H{"error": "Failed to hash password"})
		return
	}

	user := User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "user",
		IsActive:     true,
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(400, goTap.H{"error": "Failed to create user"})
		return
	}

	c.JSON(201, goTap.H{
		"message": "User created successfully",
		"user": goTap.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// login godoc
// @Summary      User login
// @Description  Authenticate user and return JWT token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200  {object}  SuccessResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /auth/login [post]
func login(c *goTap.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	var user User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(401, goTap.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(401, goTap.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	claims := goTap.JWTClaims{
		UserID:    fmt.Sprint(user.ID),
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	token, err := goTap.GenerateJWT(jwtSecret, claims)
	if err != nil {
		c.JSON(500, goTap.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(200, goTap.H{
		"token": token,
		"user": goTap.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// getProfile godoc
// @Summary      Get user profile
// @Description  Get current authenticated user's profile
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  User
// @Failure      401  {object}  ErrorResponse
// @Router       /auth/profile [get]
func getProfile(c *goTap.Context) {
	claims, _ := goTap.GetJWTClaims(c)
	userID, _ := strconv.ParseUint(claims.UserID, 10, 32)

	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(404, goTap.H{"error": "User not found"})
		return
	}

	c.JSON(200, user)
}

// getProducts godoc
// @Summary      List products
// @Description  Get paginated list of products
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page     query  int     false  "Page number"  default(1)
// @Param        per_page query  int     false  "Items per page"  default(10)
// @Param        category query  string  false  "Filter by category"
// @Success      200  {object}  SuccessResponse{data=[]Product}
// @Failure      401  {object}  ErrorResponse
// @Router       /products [get]
func getProducts(c *goTap.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	category := c.Query("category")

	offset := (page - 1) * perPage

	query := db.Model(&Product{})
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var total int64
	query.Count(&total)

	var products []Product
	query.Offset(offset).Limit(perPage).Find(&products)

	c.JSON(200, goTap.H{
		"products": products,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// getProduct godoc
// @Summary      Get product by ID
// @Description  Get detailed information about a product
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  Product
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /products/{id} [get]
func getProduct(c *goTap.Context) {
	id := c.Param("id")

	var product Product
	if err := db.First(&product, id).Error; err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	c.JSON(200, product)
}

// createProduct godoc
// @Summary      Create product
// @Description  Create a new product (admin only)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateProductRequest true "Product data"
// @Success      201  {object}  Product
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Router       /admin/products [post]
func createProduct(c *goTap.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	product := Product{
		Name:        req.Name,
		SKU:         req.SKU,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
	}

	if err := db.Create(&product).Error; err != nil {
		c.JSON(400, goTap.H{"error": "Failed to create product"})
		return
	}

	c.JSON(201, product)
}

// updateProduct godoc
// @Summary      Update product
// @Description  Update an existing product (admin only)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                  true  "Product ID"
// @Param        request  body      CreateProductRequest true  "Product data"
// @Success      200      {object}  Product
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Router       /admin/products/{id} [put]
func updateProduct(c *goTap.Context) {
	id := c.Param("id")

	var product Product
	if err := db.First(&product, id).Error; err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	product.Name = req.Name
	product.SKU = req.SKU
	product.Description = req.Description
	product.Price = req.Price
	product.Stock = req.Stock
	product.Category = req.Category

	db.Save(&product)
	c.JSON(200, product)
}

// deleteProduct godoc
// @Summary      Delete product
// @Description  Delete a product (admin only)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  SuccessResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /admin/products/{id} [delete]
func deleteProduct(c *goTap.Context) {
	id := c.Param("id")

	if err := db.Delete(&Product{}, id).Error; err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	c.JSON(200, goTap.H{"message": "Product deleted successfully"})
}

// listUsers godoc
// @Summary      List all users
// @Description  Get list of all users (admin only)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  SuccessResponse{data=[]User}
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Router       /admin/users [get]
func listUsers(c *goTap.Context) {
	var users []User
	db.Find(&users)

	c.JSON(200, goTap.H{
		"users": users,
		"count": len(users),
	})
}

func seedData() {
	// Seed admin user
	var count int64
	db.Model(&User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 14)
		admin := User{
			Username:     "admin",
			Email:        "admin@example.com",
			PasswordHash: string(hash),
			Role:         "admin",
			IsActive:     true,
		}
		db.Create(&admin)
		log.Println("✅ Default admin created")
	}

	// Seed sample products
	db.Model(&Product{}).Count(&count)
	if count == 0 {
		products := []Product{
			{Name: "Laptop", SKU: "LAP001", Description: "High-performance laptop", Price: 999.99, Stock: 10, Category: "Electronics"},
			{Name: "Mouse", SKU: "MOU001", Description: "Wireless mouse", Price: 29.99, Stock: 50, Category: "Accessories"},
			{Name: "Keyboard", SKU: "KEY001", Description: "Mechanical keyboard", Price: 79.99, Stock: 30, Category: "Accessories"},
		}
		db.Create(&products)
		log.Println("✅ Sample products seeded")
	}
}
