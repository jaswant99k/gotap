#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Creates a new goTap project with GORM, Redis, and MongoDB support.

.DESCRIPTION
    This script scaffolds a complete goTap project with:
    - Go module initialization
    - Local goTap framework linking
    - Database driver installation (MySQL/PostgreSQL)
    - Sample models and CRUD endpoints
    - Configuration file
    - Optional Redis and MongoDB support

.PARAMETER ProjectPath
    The full path where the project will be created.

.PARAMETER ProjectName
    The name of the project (used for go module name).

.PARAMETER Database
    The database driver to install: mysql, postgres, or sqlite (default: mysql).

.PARAMETER GoTapPath
    Path to local goTap framework (default: C:\goTap).

.PARAMETER IncludeRedis
    Include Redis middleware and dependencies.

.PARAMETER IncludeMongo
    Include MongoDB middleware and dependencies.

.EXAMPLE
    .\new-project.ps1 -ProjectPath "C:\Users\verve\Music\vervepos" -ProjectName "vervepos"

.EXAMPLE
    .\new-project.ps1 -ProjectPath "C:\projects\myapi" -ProjectName "myapi" -Database postgres -IncludeRedis

#>

param(
    [Parameter(Mandatory=$true, HelpMessage="Full path to create the project")]
    [string]$ProjectPath,
    
    [Parameter(Mandatory=$false, HelpMessage="Project name for go module")]
    [string]$ProjectName = "",
    
    [Parameter(Mandatory=$false, HelpMessage="Database driver: mysql, postgres, or sqlite")]
    [ValidateSet("mysql", "postgres", "sqlite")]
    [string]$Database = "mysql",
    
    [Parameter(Mandatory=$false, HelpMessage="Path to local goTap framework")]
    [string]$GoTapPath = "C:\goTap",
    
    [Parameter(Mandatory=$false, HelpMessage="Include Redis support")]
    [switch]$IncludeRedis,
    
    [Parameter(Mandatory=$false, HelpMessage="Include MongoDB support")]
    [switch]$IncludeMongo,
    
    [Parameter(Mandatory=$false, HelpMessage="Use modular structure (feature-based)")]
    [switch]$Modular
)

# Colors for output
function Write-Success { param($Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Info { param($Message) Write-Host "ℹ️  $Message" -ForegroundColor Cyan }
function Write-Warning { param($Message) Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Error-Custom { param($Message) Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Step { param($Message) Write-Host "`n🔹 $Message" -ForegroundColor Blue }

# Banner
Write-Host @"

╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║         goTap Project Generator v1.0                      ║
║         High-Performance Web Framework for Go             ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

"@ -ForegroundColor Magenta

# Extract project name from path if not provided
if ([string]::IsNullOrEmpty($ProjectName)) {
    $ProjectName = Split-Path -Leaf $ProjectPath
    Write-Info "Using project name: $ProjectName"
}

# Validate goTap path
if (-not (Test-Path $GoTapPath)) {
    Write-Error-Custom "goTap framework not found at: $GoTapPath"
    Write-Info "Please specify correct path using -GoTapPath parameter"
    exit 1
}

# Step 1: Create project directory
Write-Step "Creating project directory..."
if (Test-Path $ProjectPath) {
    Write-Warning "Directory already exists: $ProjectPath"
    $response = Read-Host "Continue anyway? (y/N)"
    if ($response -ne "y") {
        Write-Info "Aborted."
        exit 0
    }
} else {
    New-Item -ItemType Directory -Path $ProjectPath -Force | Out-Null
    Write-Success "Created directory: $ProjectPath"
}

Set-Location $ProjectPath

# Step 2: Initialize Go module
Write-Step "Initializing Go module..."
go mod init $ProjectName
if ($LASTEXITCODE -eq 0) {
    Write-Success "Initialized Go module: $ProjectName"
} else {
    Write-Error-Custom "Failed to initialize Go module"
    exit 1
}

# Step 3: Add local goTap replace directive
Write-Step "Linking local goTap framework..."
go mod edit -replace "github.com/yourusername/goTap=$GoTapPath"
Write-Success "Linked to local goTap: $GoTapPath"

# Step 4: Install dependencies
Write-Step "Installing dependencies..."

Write-Info "Installing goTap framework..."
go get github.com/yourusername/goTap

Write-Info "Installing GORM..."
go get gorm.io/gorm

# Database driver
switch ($Database) {
    "mysql" {
        Write-Info "Installing MySQL driver..."
        go get gorm.io/driver/mysql
        $dsnExample = "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
    }
    "postgres" {
        Write-Info "Installing PostgreSQL driver..."
        go get gorm.io/driver/postgres
        $dsnExample = "host=localhost user=user password=password dbname=dbname port=5432 sslmode=disable"
    }
    "sqlite" {
        Write-Info "Installing SQLite driver..."
        go get gorm.io/driver/sqlite
        $dsnExample = "database.db"
    }
}

# Optional dependencies
if ($IncludeRedis) {
    Write-Info "Installing Redis client..."
    go get github.com/redis/go-redis/v9
}

if ($IncludeMongo) {
    Write-Info "Installing MongoDB driver..."
    go get go.mongodb.org/mongo-driver/mongo
}

go mod tidy
Write-Success "Dependencies installed"

# Step 5: Create directory structure
Write-Step "Creating project structure..."

if ($Modular) {
    # Modular (feature-based) structure
    $directories = @(
        "cmd/server",
        "modules/auth",
        "modules/products",
        "modules/customers",
        "modules/orders",
        "shared/database",
        "shared/middleware",
        "shared/utils",
        "config"
    )
    Write-Info "Using modular (feature-based) structure"
} else {
    # Traditional layer-based structure
    $directories = @(
        "models",
        "handlers",
        "middleware",
        "config",
        "utils"
    )
    Write-Info "Using layer-based structure"
}

foreach ($dir in $directories) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
}
Write-Success "Created project structure"

# Step 6: Generate configuration file
Write-Step "Generating configuration file..."
$configContent = @"
# $ProjectName Configuration

# Server
SERVER_PORT=8080
SERVER_HOST=localhost

# Database
DB_DRIVER=$Database
DB_DSN=$dsnExample

# Connection Pool
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=100
DB_CONN_MAX_LIFETIME=3600

# Redis (if enabled)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# MongoDB (if enabled)
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=$ProjectName

# JWT
JWT_SECRET=your-secret-key-change-this-in-production

# Environment
ENVIRONMENT=development
"@

$configContent | Out-File -FilePath "config/.env.example" -Encoding UTF8
$configContent | Out-File -FilePath "config/.env" -Encoding UTF8
Write-Success "Created config/.env"

# Step 7: Generate models
Write-Step "Generating sample models..."
$modelsContent = @"
package models

import (
	"gorm.io/gorm"
)

// Product represents a product in the system
type Product struct {
	gorm.Model
	Name        string  ``gorm:"not null;size:255" json:"name" binding:"required"``
	SKU         string  ``gorm:"uniqueIndex;not null;size:100" json:"sku" binding:"required"``
	Price       float64 ``gorm:"type:decimal(10,2);not null" json:"price" binding:"required,gt=0"``
	Stock       int     ``gorm:"default:0" json:"stock" binding:"gte=0"``
	Category    string  ``gorm:"index;size:100" json:"category"``
	Description string  ``gorm:"type:text" json:"description"``
}

// Customer represents a customer
type Customer struct {
	gorm.Model
	Name          string ``gorm:"not null;size:255" json:"name" binding:"required"``
	Email         string ``gorm:"uniqueIndex;not null;size:255" json:"email" binding:"required,email"``
	Phone         string ``gorm:"size:20" json:"phone"``
	LoyaltyPoints int    ``gorm:"default:0" json:"loyalty_points"``
}

// Order represents a customer order
type Order struct {
	gorm.Model
	CustomerID    uint        ``gorm:"index;not null" json:"customer_id" binding:"required"``
	Customer      Customer    ``json:"customer,omitempty"``
	Total         float64     ``gorm:"type:decimal(10,2);not null" json:"total"``
	Status        string      ``gorm:"size:50;default:'pending'" json:"status"``
	PaymentMethod string      ``gorm:"size:50" json:"payment_method"``
	Items         []OrderItem ``gorm:"foreignKey:OrderID" json:"items,omitempty"``
}

// OrderItem represents an item in an order
type OrderItem struct {
	gorm.Model
	OrderID   uint    ``gorm:"index;not null" json:"order_id"``
	ProductID uint    ``gorm:"index;not null" json:"product_id" binding:"required"``
	Product   Product ``json:"product,omitempty"``
	Quantity  int     ``gorm:"not null" json:"quantity" binding:"required,gt=0"``
	Price     float64 ``gorm:"type:decimal(10,2);not null" json:"price"``
}
"@

$modelsContent | Out-File -FilePath "models/models.go" -Encoding UTF8
Write-Success "Created models/models.go"

# Step 8: Generate handlers
Write-Step "Generating handlers..."
$handlersContent = @"
package handlers

import (
	"$ProjectName/models"
	"github.com/yourusername/goTap"
)

// ProductHandlers contains all product-related handlers
type ProductHandlers struct{}

// GetProducts returns all products with pagination
func (h *ProductHandlers) GetProducts(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	pagination := goTap.NewGormPagination(c)

	var products []models.Product
	var total int64

	query := db.Model(&models.Product{})

	// Filter by category if provided
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	// Search by name or SKU
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR sku LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)
	pagination.Apply(query).Find(&products)

	c.JSON(200, goTap.H{
		"data":      products,
		"page":      pagination.Page,
		"page_size": pagination.PageSize,
		"total":     total,
	})
}

// GetProduct returns a single product by ID
func (h *ProductHandlers) GetProduct(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	var product models.Product

	if err := goTap.GormFindByID(db, &product, c.Param("id")); err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	c.JSON(200, product)
}

// CreateProduct creates a new product
func (h *ProductHandlers) CreateProduct(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	var product models.Product

	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	// Check if SKU already exists
	exists, _ := goTap.GormExists(db, &models.Product{}, "sku = ?", product.SKU)
	if exists {
		c.JSON(400, goTap.H{"error": "SKU already exists"})
		return
	}

	if err := goTap.GormCreate(db, &product); err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(201, product)
}

// UpdateProduct updates an existing product
func (h *ProductHandlers) UpdateProduct(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	var product models.Product

	if err := goTap.GormFindByID(db, &product, c.Param("id")); err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	var input models.Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	// Check SKU uniqueness if changed
	if input.SKU != product.SKU {
		exists, _ := goTap.GormExists(db, &models.Product{}, "sku = ? AND id != ?", input.SKU, c.Param("id"))
		if exists {
			c.JSON(400, goTap.H{"error": "SKU already exists"})
			return
		}
	}

	updates := map[string]interface{}{
		"name":        input.Name,
		"sku":         input.SKU,
		"price":       input.Price,
		"stock":       input.Stock,
		"category":    input.Category,
		"description": input.Description,
	}

	goTap.GormUpdate(db, &product, updates)
	goTap.GormFindByID(db, &product, c.Param("id"))

	c.JSON(200, product)
}

// DeleteProduct soft deletes a product
func (h *ProductHandlers) DeleteProduct(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	var product models.Product

	if err := goTap.GormFindByID(db, &product, c.Param("id")); err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	goTap.GormDelete(db, &product)
	c.JSON(200, goTap.H{"message": "Product deleted successfully"})
}

// CustomerHandlers contains all customer-related handlers
type CustomerHandlers struct{}

// GetCustomers returns all customers with pagination
func (h *CustomerHandlers) GetCustomers(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	pagination := goTap.NewGormPagination(c)

	var customers []models.Customer
	var total int64

	query := db.Model(&models.Customer{})

	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)
	pagination.Apply(query).Find(&customers)

	c.JSON(200, goTap.H{
		"data":      customers,
		"page":      pagination.Page,
		"page_size": pagination.PageSize,
		"total":     total,
	})
}

// CreateCustomer creates a new customer
func (h *CustomerHandlers) CreateCustomer(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	var customer models.Customer

	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	exists, _ := goTap.GormExists(db, &models.Customer{}, "email = ?", customer.Email)
	if exists {
		c.JSON(400, goTap.H{"error": "Email already exists"})
		return
	}

	if err := goTap.GormCreate(db, &customer); err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(201, customer)
}
"@

$handlersContent | Out-File -FilePath "handlers/handlers.go" -Encoding UTF8
Write-Success "Created handlers/handlers.go"

# Step 9: Generate main.go
Write-Step "Generating main.go..."
$redisMiddleware = if ($IncludeRedis) { "router.Use(goTap.RedisInject(redisClient))" } else { "" }
$mongoMiddleware = if ($IncludeMongo) { "router.Use(goTap.MongoInject(mongoClient))" } else { "" }

$mainContent = @"
package main

import (
	"log"
	"os"
	"time"

	"$ProjectName/handlers"
	"$ProjectName/models"
	"github.com/yourusername/goTap"
	"gorm.io/gorm/logger"
)

func main() {
	log.Println("Starting $ProjectName server...")

	// Load environment variables (you can use godotenv package)
	dbDriver := getEnv("DB_DRIVER", "$Database")
	dbDSN := getEnv("DB_DSN", "$dsnExample")
	serverPort := getEnv("SERVER_PORT", "8080")

	// Database configuration
	dbConfig := &goTap.DBConfig{
		Driver:          dbDriver,
		DSN:             dbDSN,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		LogLevel:        logger.Info,
	}

	// Connect to database
	db, err := goTap.NewGormDB(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	log.Println("✅ Database connected successfully")

	// Auto-migrate models
	if err := goTap.AutoMigrate(db, 
		&models.Product{}, 
		&models.Customer{}, 
		&models.Order{}, 
		&models.OrderItem{},
	); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Println("✅ Database migrated successfully")

	// Create router
	router := goTap.New()

	// Global middleware
	router.Use(goTap.Logger())
	router.Use(goTap.Recovery())
	router.Use(goTap.CORS())
	router.Use(goTap.GormInject(db))
	$redisMiddleware
	$mongoMiddleware

	// Health check
	router.GET("/health", goTap.GormHealthCheck())

	// Initialize handlers
	productHandlers := &handlers.ProductHandlers{}
	customerHandlers := &handlers.CustomerHandlers{}

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Products
		products := v1.Group("/products")
		{
			products.GET("", productHandlers.GetProducts)
			products.GET("/:id", productHandlers.GetProduct)
			products.POST("", productHandlers.CreateProduct)
			products.PUT("/:id", productHandlers.UpdateProduct)
			products.DELETE("/:id", productHandlers.DeleteProduct)
		}

		// Customers
		customers := v1.Group("/customers")
		{
			customers.GET("", customerHandlers.GetCustomers)
			customers.POST("", customerHandlers.CreateCustomer)
		}
	}

	// Start server
	log.Printf("✅ Server starting on http://localhost:%s", serverPort)
	log.Println("📚 API Documentation:")
	log.Println("   GET    /health")
	log.Println("   GET    /api/v1/products")
	log.Println("   GET    /api/v1/products/:id")
	log.Println("   POST   /api/v1/products")
	log.Println("   PUT    /api/v1/products/:id")
	log.Println("   DELETE /api/v1/products/:id")
	log.Println("   GET    /api/v1/customers")
	log.Println("   POST   /api/v1/customers")
	
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
"@

$mainContent | Out-File -FilePath "main.go" -Encoding UTF8
Write-Success "Created main.go"

# Step 10: Create README
Write-Step "Generating README.md..."
$readmeContent = @"
# $ProjectName

A high-performance REST API built with [goTap](https://github.com/jaswant99k/gotap) framework.

## Features

- ✅ RESTful API with CRUD operations
- ✅ GORM ORM with $Database support
- ✅ Auto-migration
- ✅ Pagination
- ✅ Input validation
- ✅ Soft deletes
- ✅ CORS enabled
- ✅ Health check endpoint
$(if ($IncludeRedis) { "- ✅ Redis caching" } else { "" })
$(if ($IncludeMongo) { "- ✅ MongoDB support" } else { "" })

## Quick Start

### 1. Configure Database

Edit ``config/.env`` and set your database credentials:

``````env
DB_DSN=$dsnExample
``````

### 2. Create Database

**MySQL:**
``````sql
CREATE DATABASE $ProjectName;
CREATE USER 'user'@'localhost' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON $ProjectName.* TO 'user'@'localhost';
``````

**PostgreSQL:**
``````sql
CREATE DATABASE $ProjectName;
``````

### 3. Run Application

``````bash
go run main.go
``````

Server starts at: http://localhost:8080

## API Endpoints

### Products

``````bash
# Get all products (with pagination)
GET /api/v1/products?page=1&page_size=20

# Search products
GET /api/v1/products?search=laptop&category=electronics

# Get single product
GET /api/v1/products/:id

# Create product
POST /api/v1/products
Content-Type: application/json

{
  "name": "Laptop",
  "sku": "LAP001",
  "price": 999.99,
  "stock": 50,
  "category": "electronics"
}

# Update product
PUT /api/v1/products/:id

# Delete product (soft delete)
DELETE /api/v1/products/:id
``````

### Customers

``````bash
# Get all customers
GET /api/v1/customers

# Create customer
POST /api/v1/customers
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "phone": "+1234567890"
}
``````

### Health Check

``````bash
GET /health
``````

## Testing with curl (PowerShell)

``````powershell
# Create a product
curl -X POST http://localhost:8080/api/v1/products ``
  -H "Content-Type: application/json" ``
  -d '{\"name\":\"Laptop\",\"sku\":\"LAP001\",\"price\":999.99,\"stock\":50,\"category\":\"electronics\"}'

# Get all products
curl http://localhost:8080/api/v1/products

# Search products
curl "http://localhost:8080/api/v1/products?search=laptop"

# Get product by ID
curl http://localhost:8080/api/v1/products/1

# Update product
curl -X PUT http://localhost:8080/api/v1/products/1 ``
  -H "Content-Type: application/json" ``
  -d '{\"name\":\"Gaming Laptop\",\"sku\":\"LAP001\",\"price\":1299.99,\"stock\":45,\"category\":\"electronics\"}'

# Delete product
curl -X DELETE http://localhost:8080/api/v1/products/1
``````

## Project Structure

``````
$ProjectName/
├── main.go                 # Application entry point
├── models/                 # Database models
│   └── models.go
├── handlers/               # HTTP handlers
│   └── handlers.go
├── middleware/             # Custom middleware
├── config/                 # Configuration files
│   ├── .env
│   └── .env.example
└── utils/                  # Utility functions
``````

## Development

### Add New Models

Edit ``models/models.go`` and add your struct:

``````go
type YourModel struct {
    gorm.Model
    Name string ``gorm:"not null" json:"name"``
}
``````

Then migrate:

``````go
goTap.AutoMigrate(db, &models.YourModel{})
``````

### Add New Handlers

Create handler in ``handlers/handlers.go`` and register route in ``main.go``.

## Documentation

- [goTap Framework](https://github.com/jaswant99k/gotap)
- [GORM Documentation](https://gorm.io/docs/)

## License

MIT
"@

$readmeContent | Out-File -FilePath "README.md" -Encoding UTF8
Write-Success "Created README.md"

# Step 11: Create .gitignore
Write-Step "Creating .gitignore..."
$gitignoreContent = @"
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test files
*.test
*.out

# Go workspace file
go.work

# Environment files
.env
config/.env

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Database
*.db
*.sqlite
*.sqlite3

# Logs
*.log
logs/

# Build
build/
dist/
bin/
"@

$gitignoreContent | Out-File -FilePath ".gitignore" -Encoding UTF8
Write-Success "Created .gitignore"

# Summary
Write-Host "`n" -NoNewline
Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Magenta
Write-Host "                    🎉 Success!                            " -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Magenta
Write-Host ""
Write-Success "Project created at: $ProjectPath"
Write-Success "Project name: $ProjectName"
Write-Success "Database: $Database"
if ($IncludeRedis) { Write-Success "Redis: Enabled" }
if ($IncludeMongo) { Write-Success "MongoDB: Enabled" }

Write-Host "`n📋 Next Steps:" -ForegroundColor Cyan
Write-Host "   1. cd $ProjectPath" -ForegroundColor White
Write-Host "   2. Edit config/.env with your database credentials" -ForegroundColor White
Write-Host "   3. Create the database in $Database" -ForegroundColor White
Write-Host "   4. go run main.go" -ForegroundColor White
Write-Host "   5. Open http://localhost:8080/health" -ForegroundColor White

Write-Host "`n📚 Documentation:" -ForegroundColor Cyan
Write-Host "   - README.md in your project directory" -ForegroundColor White
Write-Host "   - $GoTapPath\examples\gorm\README.md" -ForegroundColor White
Write-Host "   - $GoTapPath\GORM_QUICK_REFERENCE.md" -ForegroundColor White

Write-Host "`n🚀 Happy Coding with goTap!" -ForegroundColor Magenta
Write-Host ""
