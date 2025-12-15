#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Creates a new goTap project with modular (feature-based) structure.

.DESCRIPTION
    This script scaffolds a complete goTap project with modular architecture where
    each feature (auth, products, orders) is self-contained in its own module directory.

.PARAMETER ProjectPath
    The full path where the project will be created.

.PARAMETER ProjectName
    The name of the project (used for go module name).

.PARAMETER Database
    The database driver to install: mysql, postgres, or sqlite (default: postgres).

.PARAMETER GoTapPath
    Path to local goTap framework (optional, for development).

.PARAMETER UseLocal
    Use local goTap instead of published module from GitHub.

.EXAMPLE
    .\new-modular-project.ps1 -ProjectPath "C:\projects\vervepos" -ProjectName "vervepos"

.EXAMPLE
    .\new-modular-project.ps1 -ProjectPath "C:\projects\myapp" -ProjectName "myapp" -Database mysql

.EXAMPLE
    .\new-modular-project.ps1 -ProjectPath "C:\projects\myapp" -UseLocal -GoTapPath "C:\goTap"

#>

param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectPath,
    
    [Parameter(Mandatory=$false)]
    [string]$ProjectName = "",
    
    [Parameter(Mandatory=$false)]
    [ValidateSet("mysql", "postgres", "sqlite")]
    [string]$Database = "postgres",
    
    [Parameter(Mandatory=$false)]
    [string]$GoTapPath = "",
    
    [Parameter(Mandatory=$false)]
    [switch]$UseLocal = $false
)

# Colors
function Write-Success { param($Message) Write-Host "[OK] $Message" -ForegroundColor Green }
function Write-Info { param($Message) Write-Host "[INFO] $Message" -ForegroundColor Cyan }
function Write-Warning { param($Message) Write-Host "[WARN] $Message" -ForegroundColor Yellow }
function Write-Error-Custom { param($Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }
function Write-Step { param($Message) Write-Host "`n[STEP] $Message" -ForegroundColor Blue }

# Banner
Write-Host @"


                                                            
      goTap Modular Project Generator v2.0                 
      Feature-Based Architecture for Scalable Apps         
                                                            


"@ -ForegroundColor Magenta

# Check Go version
Write-Step "Checking Go installation..."
$goVersion = go version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Error-Custom "Go is not installed or not in PATH"
    Write-Info "Please install Go 1.22 or later from https://go.dev/dl/"
    exit 1
}
Write-Info "Found: $goVersion"

# Check if using local goTap or published module
if ($UseLocal) {
    if ([string]::IsNullOrEmpty($GoTapPath)) {
        $GoTapPath = "C:\goTap"
    }
    if (-not (Test-Path $GoTapPath)) {
        Write-Error-Custom "Local goTap path not found: $GoTapPath"
        Write-Info "Either fix the path or run without -UseLocal to use published module"
        exit 1
    }
    Write-Info "Using local goTap from: $GoTapPath"
} else {
    Write-Info "Using published goTap module (github.com/jaswant99k/gotap@v0.1.0)"
}

# Extract project name if not provided
if ([string]::IsNullOrEmpty($ProjectName)) {
    $ProjectName = Split-Path -Leaf $ProjectPath
}

Write-Info "Project: $ProjectName"
Write-Info "Location: $ProjectPath"
Write-Info "Database: $Database"
Write-Info "Structure: Modular feature-based"

# Create project directory
if (Test-Path $ProjectPath) {
    $response = Read-Host "Directory exists. Overwrite? (y/N)"
    if ($response -ne 'y') {
        Write-Warning "Cancelled"
        exit 0
    }
    Remove-Item -Path $ProjectPath -Recurse -Force
}

Write-Step "Creating project directory..."
New-Item -ItemType Directory -Path $ProjectPath -Force | Out-Null
Set-Location $ProjectPath
Write-Success "Created $ProjectPath"

# Initialize Go module
Write-Step "Initializing Go module..."
$goModContent = @"
module $ProjectName

go 1.23
"@
[System.IO.File]::WriteAllText("$ProjectPath\go.mod", $goModContent)
Write-Success "Go module initialized with Go 1.23"

# Add goTap replace directive only if using local
if ($UseLocal) {
    Write-Step "Linking local goTap framework..."
    Add-Content -Path "go.mod" -Value "`nreplace github.com/jaswant99k/gotap => $GoTapPath"
    Write-Success "Linked goTap from $GoTapPath"
}

# Install dependencies
Write-Step "Installing dependencies..."
if ($UseLocal) {
    go get github.com/jaswant99k/gotap 2>&1 | Out-Null
} else {
    go get github.com/jaswant99k/gotap@v0.1.0 2>&1 | Out-Null
}
go get gorm.io/gorm@v1.25.12 2>&1 | Out-Null
go get github.com/swaggo/files@v1.0.1 2>&1 | Out-Null
go get github.com/swaggo/gin-swagger@v1.6.0 2>&1 | Out-Null
go get github.com/swaggo/swag@v1.16.4 2>&1 | Out-Null

switch ($Database) {
    "mysql" {
        go get gorm.io/driver/mysql@v1.5.7 2>&1 | Out-Null
        $dsnExample = "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
    }
    "postgres" {
        go get gorm.io/driver/postgres@v1.5.9 2>&1 | Out-Null
        $dsnExample = "host=localhost user=postgres password=postgres dbname=$ProjectName port=5432 sslmode=disable"
    }
    "sqlite" {
        go get gorm.io/driver/sqlite@v1.5.6 2>&1 | Out-Null
        $dsnExample = "database.db"
    }
}

Write-Info "Running go mod tidy to resolve all dependencies..."
go mod tidy
if ($LASTEXITCODE -eq 0) {
    Write-Success "Dependencies installed successfully"
} else {
    Write-Warning "Some dependencies may need manual installation"
    Write-Info "Run 'go mod tidy' in the project directory to resolve"
}

# Create modular structure
Write-Step "Creating modular directory structure..."
$directories = @(
    "cmd/server",
    "modules/auth",
    "modules/products",
    "modules/customers",
    "modules/orders",
    "shared/database",
    "shared/middleware",
    "shared/utils",
    "config",
    "docs"
)

foreach ($dir in $directories) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
}
Write-Success "Created modular structure"

# Generate .env file
Write-Step "Generating configuration..."
$envContent = @"
# $ProjectName Configuration

# Server
SERVER_PORT=8080

# Database
DB_DSN=$dsnExample
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=100

# JWT Secret (change in production!)
JWT_SECRET=your-super-secret-jwt-key-minimum-32-characters-long-change-this

# Logging
LOG_LEVEL=info
"@

$envContent | Out-File -FilePath "config/.env" -Encoding UTF8
$envContent | Out-File -FilePath "config/.env.example" -Encoding UTF8
Write-Success "Created config files"

# Generate shared/database/connection.go
Write-Step "Generating shared modules..."
$dbContent = @"
package database

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/$Database"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect() (*gorm.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is required")
	}

	db, err := gorm.Open($Database.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println(" Database connected")
	return db, nil
}
"@

$dbContent | Out-File -FilePath "shared/database/connection.go" -Encoding UTF8
Write-Success "Created shared/database/connection.go"

# Generate modules/auth/models.go
$authModelsContent = @"
package auth

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username     string       ``gorm:"uniqueIndex;not null" json:"username"``
	Email        string       ``gorm:"uniqueIndex;not null" json:"email"``
	PasswordHash string       ``gorm:"not null" json:"-"``
	Role         string       ``gorm:"default:'user'" json:"role"``
	IsActive     bool         ``gorm:"default:true" json:"is_active"``
	Permissions  []Permission ``gorm:"many2many:user_permissions;" json:"permissions,omitempty"``
}

type Permission struct {
	gorm.Model
	Name        string ``gorm:"uniqueIndex;not null" json:"name"``
	Description string ``json:"description"``
	Users       []User ``gorm:"many2many:user_permissions;" json:"-"``
}

type LoginRequest struct {
	Email    string ``json:"email" binding:"required,email"``
	Password string ``json:"password" binding:"required"``
}

type RegisterRequest struct {
	Username string ``json:"username" binding:"required,min=3"``
	Email    string ``json:"email" binding:"required,email"``
	Password string ``json:"password" binding:"required,min=8"``
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

func (u *User) HasPermission(permissionName string) bool {
	for _, perm := range u.Permissions {
		if perm.Name == permissionName {
			return true
		}
	}
	return false
}
"@

$authModelsContent | Out-File -FilePath "modules/auth/models.go" -Encoding UTF8
Write-Success "Created modules/auth/models.go"

# Generate modules/auth/repository.go
$authRepoContent = @"
package auth

import "gorm.io/gorm"

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(user *User) (*User, error) {
	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) FindByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ?", email).Preload("Permissions").First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindByID(id uint) (*User, error) {
	var user User
	if err := r.db.Preload("Permissions").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindAll() ([]User, error) {
	var users []User
	if err := r.db.Preload("Permissions").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
"@

$authRepoContent | Out-File -FilePath "modules/auth/repository.go" -Encoding UTF8
Write-Success "Created modules/auth/repository.go"

# Generate modules/auth/service.go
$authServiceContent = @"
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/jaswant99k/gotap"
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) Register(req RegisterRequest) (*User, error) {
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         "user",
		IsActive:     true,
	}

	return s.repo.Create(user)
}

func (s *Service) Login(req LoginRequest) (string, *User, error) {
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return "", nil, errors.New("account is deactivated")
	}

	if !user.VerifyPassword(req.Password) {
		return "", nil, errors.New("invalid credentials")
	}

	claims := goTap.JWTClaims{
		UserID:    fmt.Sprint(user.ID),
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	token, err := goTap.GenerateJWT(s.jwtSecret, claims)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *Service) GetUserByID(id uint) (*User, error) {
	return s.repo.FindByID(id)
}

func (s *Service) GetAllUsers() ([]User, error) {
	return s.repo.FindAll()
}
"@

$authServiceContent | Out-File -FilePath "modules/auth/service.go" -Encoding UTF8
Write-Success "Created modules/auth/service.go"

# Generate modules/auth/handlers.go
$authHandlersContent = @"
package auth

import (
	"strconv"

	"github.com/jaswant99k/gotap"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register godoc
// @Summary      Register new user
// @Description  Create a new user account
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration data"
// @Success      201  {object}  map[string]interface{}  "User created successfully"
// @Failure      400  {object}  map[string]interface{}  "Bad request"
// @Router       /auth/register [post]
func (h *Handler) Register(c *goTap.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
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

// Login godoc
// @Summary      User login
// @Description  Authenticate user and return JWT token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200  {object}  map[string]interface{}  "Login successful"
// @Failure      401  {object}  map[string]interface{}  "Invalid credentials"
// @Router       /auth/login [post]
func (h *Handler) Login(c *goTap.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	token, user, err := h.service.Login(req)
	if err != nil {
		c.JSON(401, goTap.H{"error": err.Error()})
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

// GetProfile godoc
// @Summary      Get user profile
// @Description  Get current authenticated user's profile
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "User profile"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      404  {object}  map[string]interface{}  "User not found"
// @Router       /auth/profile [get]
func (h *Handler) GetProfile(c *goTap.Context) {
	claims, exists := goTap.GetJWTClaims(c)
	if !exists {
		c.JSON(401, goTap.H{"error": "Unauthorized"})
		return
	}

	userID, _ := strconv.ParseUint(claims.UserID, 10, 32)
	user, err := h.service.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(404, goTap.H{"error": "User not found"})
		return
	}

	c.JSON(200, goTap.H{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"role":        user.Role,
		"permissions": user.Permissions,
		"created_at":  user.CreatedAt,
	})
}

// ListUsers godoc
// @Summary      List all users
// @Description  Get list of all users (admin only)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "Users list"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      403  {object}  map[string]interface{}  "Forbidden"
// @Router       /admin/users [get]
func (h *Handler) ListUsers(c *goTap.Context) {
	users, err := h.service.GetAllUsers()
	if err != nil {
		c.JSON(500, goTap.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(200, goTap.H{
		"users": users,
		"count": len(users),
	})
}
"@

$authHandlersContent | Out-File -FilePath "modules/auth/handlers.go" -Encoding UTF8
Write-Success "Created modules/auth/handlers.go"

# Generate modules/auth/routes.go
$authRoutesContent = @"
package auth

import "github.com/jaswant99k/gotap"

func RegisterRoutes(r *goTap.Engine, handler *Handler, jwtSecret string) {
	// Public routes
	public := r.Group("/api/auth")
	{
		public.POST("/register", handler.Register)
		public.POST("/login", handler.Login)
	}

	// Protected routes
	protected := r.Group("/api/auth")
	protected.Use(goTap.JWTAuth(jwtSecret))
	{
		protected.GET("/profile", handler.GetProfile)
	}

	// Admin routes
	admin := r.Group("/api/admin")
	admin.Use(goTap.JWTAuth(jwtSecret))
	admin.Use(goTap.RequireRole("admin"))
	{
		admin.GET("/users", handler.ListUsers)
	}
}
"@

$authRoutesContent | Out-File -FilePath "modules/auth/routes.go" -Encoding UTF8
Write-Success "Created modules/auth/routes.go"

# Generate modules/products/models.go
$productsModelsContent = @"
package products

import "gorm.io/gorm"

type Product struct {
	gorm.Model
	Name        string  ``gorm:"not null" json:"name"``
	SKU         string  ``gorm:"uniqueIndex;not null" json:"sku"``
	Description string  ``json:"description"``
	Price       float64 ``gorm:"not null" json:"price"``
	Stock       int     ``gorm:"default:0" json:"stock"``
	Category    string  ``gorm:"index" json:"category"``
	IsActive    bool    ``gorm:"default:true" json:"is_active"``
}

type CreateProductRequest struct {
	Name        string  ``json:"name" binding:"required"``
	SKU         string  ``json:"sku" binding:"required"``
	Description string  ``json:"description"``
	Price       float64 ``json:"price" binding:"required,gt=0"``
	Stock       int     ``json:"stock" binding:"gte=0"``
	Category    string  ``json:"category"``
}
"@

$productsModelsContent | Out-File -FilePath "modules/products/models.go" -Encoding UTF8
Write-Success "Created modules/products/models.go"

# Generate modules/products/repository.go
$productsRepoContent = @"
package products

import "gorm.io/gorm"

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(product *Product) (*Product, error) {
	if err := r.db.Create(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func (r *Repository) FindAll(page, limit int) ([]Product, int64, error) {
	var products []Product
	var total int64

	offset := (page - 1) * limit

	if err := r.db.Model(&Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *Repository) FindByID(id uint) (*Product, error) {
	var product Product
	if err := r.db.First(&product, id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *Repository) FindBySKU(sku string) (*Product, error) {
	var product Product
	if err := r.db.Where("sku = ?", sku).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *Repository) Update(product *Product) error {
	return r.db.Save(product).Error
}

func (r *Repository) Delete(id uint) error {
	return r.db.Delete(&Product{}, id).Error
}
"@

$productsRepoContent | Out-File -FilePath "modules/products/repository.go" -Encoding UTF8
Write-Success "Created modules/products/repository.go"

# Generate modules/products/service.go
$productsServiceContent = @"
package products

import "errors"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateProduct(req CreateProductRequest) (*Product, error) {
	// Check if SKU already exists
	if _, err := s.repo.FindBySKU(req.SKU); err == nil {
		return nil, errors.New("SKU already exists")
	}

	product := &Product{
		Name:        req.Name,
		SKU:         req.SKU,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		IsActive:    true,
	}

	return s.repo.Create(product)
}

func (s *Service) GetAllProducts(page, limit int) ([]Product, int64, error) {
	return s.repo.FindAll(page, limit)
}

func (s *Service) GetProductByID(id uint) (*Product, error) {
	return s.repo.FindByID(id)
}

func (s *Service) UpdateProduct(id uint, req CreateProductRequest) (*Product, error) {
	product, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.Stock = req.Stock
	product.Category = req.Category

	if err := s.repo.Update(product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *Service) DeleteProduct(id uint) error {
	return s.repo.Delete(id)
}
"@

$productsServiceContent | Out-File -FilePath "modules/products/service.go" -Encoding UTF8
Write-Success "Created modules/products/service.go"

# Generate modules/products/handlers.go
$productsHandlersContent = @"
package products

import (
	"strconv"

	"github.com/jaswant99k/gotap"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateProduct godoc
// @Summary      Create product
// @Description  Create a new product (admin only)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateProductRequest true "Product data"
// @Success      201  {object}  Product
// @Failure      400  {object}  map[string]interface{}  "Bad request"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      403  {object}  map[string]interface{}  "Forbidden"
// @Router       /products [post]
func (h *Handler) CreateProduct(c *goTap.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	product, err := h.service.CreateProduct(req)
	if err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(201, product)
}

// GetProducts godoc
// @Summary      List products
// @Description  Get paginated list of products
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page   query  int  false  "Page number"  default(1)
// @Param        limit  query  int  false  "Items per page"  default(20)
// @Success      200  {object}  map[string]interface{}  "Products list"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Router       /products [get]
func (h *Handler) GetProducts(c *goTap.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	products, total, err := h.service.GetAllProducts(page, limit)
	if err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, goTap.H{
		"products": products,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetProduct godoc
// @Summary      Get product by ID
// @Description  Get detailed information about a product
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  Product
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      404  {object}  map[string]interface{}  "Product not found"
// @Router       /products/{id} [get]
func (h *Handler) GetProduct(c *goTap.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	product, err := h.service.GetProductByID(uint(id))
	if err != nil {
		c.JSON(404, goTap.H{"error": "Product not found"})
		return
	}

	c.JSON(200, product)
}

// UpdateProduct godoc
// @Summary      Update product
// @Description  Update an existing product (admin only)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                  true  "Product ID"
// @Param        request  body      CreateProductRequest true  "Product data"
// @Success      200      {object}  Product
// @Failure      400      {object}  map[string]interface{}  "Bad request"
// @Failure      401      {object}  map[string]interface{}  "Unauthorized"
// @Failure      403      {object}  map[string]interface{}  "Forbidden"
// @Failure      404      {object}  map[string]interface{}  "Product not found"
// @Router       /products/{id} [put]
func (h *Handler) UpdateProduct(c *goTap.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	product, err := h.service.UpdateProduct(uint(id), req)
	if err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, product)
}

// DeleteProduct godoc
// @Summary      Delete product
// @Description  Delete a product (admin only)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Product ID"
// @Success      200  {object}  map[string]interface{}  "Product deleted"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      403  {object}  map[string]interface{}  "Forbidden"
// @Failure      500  {object}  map[string]interface{}  "Internal error"
// @Router       /products/{id} [delete]
func (h *Handler) DeleteProduct(c *goTap.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.service.DeleteProduct(uint(id)); err != nil {
		c.JSON(500, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, goTap.H{"message": "Product deleted successfully"})
}
"@

$productsHandlersContent | Out-File -FilePath "modules/products/handlers.go" -Encoding UTF8
Write-Success "Created modules/products/handlers.go"

# Generate modules/products/routes.go
$productsRoutesContent = @"
package products

import "github.com/jaswant99k/gotap"

func RegisterRoutes(r *goTap.Engine, handler *Handler, jwtSecret string) {
	api := r.Group("/api/products")
	api.Use(goTap.JWTAuth(jwtSecret))
	{
		api.GET("", handler.GetProducts)
		api.GET("/:id", handler.GetProduct)

		// Admin only
		admin := api.Group("")
		admin.Use(goTap.RequireRole("admin"))
		{
			admin.POST("", handler.CreateProduct)
			admin.PUT("/:id", handler.UpdateProduct)
			admin.DELETE("/:id", handler.DeleteProduct)
		}
	}
}
"@

$productsRoutesContent | Out-File -FilePath "modules/products/routes.go" -Encoding UTF8
Write-Success "Created modules/products/routes.go"

# Generate main.go
Write-Step "Generating main application..."
$mainContent = @"
package main

import (
	"log"
	"os"

	"$ProjectName/modules/auth"
	"$ProjectName/modules/products"
	"$ProjectName/shared/database"
	"$ProjectName/docs"

	"github.com/jaswant99k/gotap"
	"gorm.io/gorm"
)

// @title           $ProjectName API
// @version         1.0
// @description     $ProjectName REST API with authentication and authorization
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@$ProjectName.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-super-secret-jwt-key-minimum-32-characters-long-change-this"
	}

	// Initialize database
	db, err := database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate all modules
	if err := db.AutoMigrate(
		&auth.User{},
		&auth.Permission{},
		&products.Product{},
	); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed data
	seedDefaultData(db)

	// Initialize goTap
	r := goTap.Default()
	r.Use(goTap.GormInject(db))

	// Determine port
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Update Swagger host dynamically to match the running port
	docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(addr)

	// Setup Swagger UI
	goTap.SetupSwagger(r, "/swagger")

	// Health check
	r.GET("/health", func(c *goTap.Context) {
		c.JSON(200, goTap.H{
			"status": "ok",
			"app":    "$ProjectName",
		})
	})

	// Initialize modules
	initAuthModule(r, db, jwtSecret)
	initProductsModule(r, db, jwtSecret)

	log.Println(" Server starting on http://localhost:" + port)
	log.Println(" Swagger UI: http://localhost:" + port + "/swagger/index.html")
	log.Println(" Default admin: admin@example.com / admin123")
	log.Println(" API Documentation:")
	log.Println("  POST /api/auth/register - Register new user")
	log.Println("  POST /api/auth/login - Login and get JWT token")
	log.Println("  GET  /api/auth/profile - Get current user profile (authenticated)")
	log.Println("  GET  /api/products - List products (authenticated)")
	log.Println("  POST /api/products - Create product (admin only)")
	log.Println()

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
	log.Println("  POST /api/products - Create product (admin only)")
	log.Println()

	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err))
	handler := auth.NewHandler(service)
	auth.RegisterRoutes(r, handler, jwtSecret)
	log.Println(" Auth module initialized")
}

func initProductsModule(r *goTap.Engine, db *gorm.DB, jwtSecret string) {
	repo := products.NewRepository(db)
	service := products.NewService(repo)
	handler := products.NewHandler(service)
	products.RegisterRoutes(r, handler, jwtSecret)
	log.Println(" Products module initialized")
}

func seedDefaultData(db *gorm.DB) {
	// Seed permissions
	permissions := []auth.Permission{
		{Name: "create_product", Description: "Create new products"},
		{Name: "edit_product", Description: "Edit existing products"},
		{Name: "delete_product", Description: "Delete products"},
		{Name: "view_orders", Description: "View all orders"},
		{Name: "manage_users", Description: "Manage user accounts"},
	}

	for _, perm := range permissions {
		db.FirstOrCreate(&perm, auth.Permission{Name: perm.Name})
	}

	// Create default admin user
	var count int64
	db.Model(&auth.User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		hash, _ := auth.HashPassword("admin123")
		admin := auth.User{
			Username:     "admin",
			Email:        "admin@example.com",
			PasswordHash: hash,
			Role:         "admin",
			IsActive:     true,
		}

		if err := db.Create(&admin).Error; err == nil {
			// Assign all permissions to admin
			var perms []auth.Permission
			db.Find(&perms)
			db.Model(&admin).Association("Permissions").Append(perms)
			log.Println(" Default admin created")
		}
	}

	// Seed sample products
	var productCount int64
	db.Model(&products.Product{}).Count(&productCount)

	if productCount == 0 {
		sampleProducts := []products.Product{
			{Name: "Laptop", SKU: "LAP001", Price: 999.99, Stock: 10, Category: "Electronics"},
			{Name: "Mouse", SKU: "MOU001", Price: 29.99, Stock: 50, Category: "Accessories"},
			{Name: "Keyboard", SKU: "KEY001", Price: 79.99, Stock: 30, Category: "Accessories"},
		}

		for _, product := range sampleProducts {
			db.Create(&product)
		}
		log.Println(" Sample products seeded")
	}
}
"@

$mainContent | Out-File -FilePath "cmd/server/main.go" -Encoding UTF8
Write-Success "Created cmd/server/main.go"

# Generate .gitignore
$gitignoreContent = @"
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test

# Output
/bin/
/dist/

# Dependencies
/vendor/

# Environment
.env
*.db
*.log

# IDE
.idea/
.vscode/
*.swp
*.swo
*~
.DS_Store

# Go workspace
go.work
go.work.sum
"@

$gitignoreContent | Out-File -FilePath ".gitignore" -Encoding UTF8
Write-Success "Created .gitignore"

# Generate README.md
$readmeContent = @"
# $ProjectName

Modular goTap application with feature-based architecture.

## Project Structure

\`\`\`
$ProjectName/
 cmd/
    server/
        main.go              # Application entry point

 modules/                      # Feature modules
    auth/                    # Authentication module
       models.go
       repository.go
       service.go
       handlers.go
       routes.go
   
    products/                # Products module
        models.go
        repository.go
        service.go
        handlers.go
        routes.go

 shared/                      # Shared utilities
    database/
    middleware/
    utils/

 config/
     .env
\`\`\`

## Quick Start

1. **Configure Database**
   \`\`\`bash
   cp config/.env.example config/.env
   # Edit config/.env with your database credentials
   \`\`\`

2. **Install Dependencies**
   \`\`\`bash
   go mod download
   \`\`\`

3. **Generate Swagger Documentation**
   \`\`\`bash
   # Install swag CLI (if not already installed)
   go install github.com/swaggo/swag/cmd/swag@latest
   
   # Generate Swagger docs
   swag init -g cmd/server/main.go --output docs
   \`\`\`

4. **Run Application**
   \`\`\`bash
   go run cmd/server/main.go
   \`\`\`

5. **Access Swagger UI**
   Open http://localhost:8080/swagger/index.html

6. **Test API**
   \`\`\`powershell
   # Login
   `$response = Invoke-RestMethod -Uri "http://localhost:8080/api/auth/login" \`
       -Method POST \`
       -ContentType "application/json" \`
       -Body '{\"email\":\"admin@example.com\",\"password\":\"admin123\"}'

   `$token = `$response.token

   # Get products
   Invoke-RestMethod -Uri "http://localhost:8080/api/products" \`
       -Headers @{Authorization="Bearer `$token"}
   \`\`\`

## API Endpoints

### Authentication
- \`POST /api/auth/register\` - Register new user
- \`POST /api/auth/login\` - Login and get JWT token
- \`GET /api/auth/profile\` - Get current user profile

### Products
- \`GET /api/products\` - List all products
- \`GET /api/products/:id\` - Get product by ID
- \`POST /api/products\` - Create product (admin only)
- \`PUT /api/products/:id\` - Update product (admin only)
- \`DELETE /api/products/:id\` - Delete product (admin only)

## Default Credentials

- **Email**: admin@example.com
- **Password**: admin123

 **Change these in production!**

## Adding New Modules

1. Create module directory: \`modules/newfeature/\`
2. Add files: \`models.go\`, \`repository.go\`, \`service.go\`, \`handlers.go\`, \`routes.go\`
3. Initialize in \`cmd/server/main.go\`

## Environment Variables

See \`config/.env.example\` for all available options.

## License

MIT
"@

$readmeContent | Out-File -FilePath "README.md" -Encoding UTF8
Write-Success "Created README.md"

# Create placeholder docs.go for initial build
Write-Step "Creating placeholder Swagger docs..."
$placeholderDocs = @"
// Package docs GENERATED BY SWAG; DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = ``{
    "swagger": "2.0",
    "info": {
        "title": "$ProjectName API",
        "version": "1.0",
        "description": "$ProjectName REST API"
    },
    "host": "localhost:8080",
    "basePath": "/api",
    "paths": {}
}``

var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/api",
	Schemes:          []string{},
	Title:            "$ProjectName API",
	Description:      "$ProjectName REST API",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
"@

$placeholderDocs | Out-File -FilePath "docs/docs.go" -Encoding UTF8
Write-Success "Created placeholder Swagger docs"

# Try to generate real Swagger docs if swag is installed
Write-Step "Attempting to generate Swagger documentation..."
$swagPath = (Get-Command swag -ErrorAction SilentlyContinue).Path
if ($swagPath) {
    Write-Info "Found swag CLI, generating documentation..."
    & swag init -g cmd/server/main.go --output docs 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Swagger documentation generated"
    } else {
        Write-Warning "Swagger generation had issues, using placeholder docs"
    }
} else {
    Write-Warning "swag CLI not found. Install it with: go install github.com/swaggo/swag/cmd/swag@latest"
    Write-Info "Using placeholder docs for now. Run 'swag init -g cmd/server/main.go --output docs' later"
}

# Final go mod tidy to ensure everything is resolved
Write-Step "Final dependency resolution..."
go mod tidy 2>&1 | Out-Null
Write-Success "Project setup complete"

# Summary
Write-Host "`n"
Write-Host "" -ForegroundColor Green
Write-Host "                                                            " -ForegroundColor Green
Write-Host "            Project Created Successfully!                 " -ForegroundColor Green
Write-Host "                                                            " -ForegroundColor Green
Write-Host "" -ForegroundColor Green
Write-Host "`n"

Write-Info "Project: $ProjectName"
Write-Info "Location: $ProjectPath"
Write-Info "Structure: Modular feature-based"
Write-Info "Database: $Database"

Write-Host "`nNext Steps:" -ForegroundColor Yellow
Write-Host "  1. cd $ProjectPath" -ForegroundColor White
Write-Host "  2. Edit config/.env with your database credentials" -ForegroundColor White
Write-Host "  3. go run cmd/server/main.go" -ForegroundColor White
Write-Host "  4. Open http://localhost:8080/swagger/index.html" -ForegroundColor White
Write-Host "`n"

if (-not $swagPath) {
    Write-Host "Optional: Generate complete Swagger docs" -ForegroundColor Yellow
    Write-Host "  1. go install github.com/swaggo/swag/cmd/swag@latest" -ForegroundColor White
    Write-Host "  2. swag init -g cmd/server/main.go --output docs" -ForegroundColor White
    Write-Host "`n"
}

Write-Info "Default admin credentials:"
Write-Host "  Email: admin@example.com" -ForegroundColor Cyan
Write-Host "  Password: admin123" -ForegroundColor Cyan
Write-Host "`n"

Write-Info "Swagger UI will be available at:"
Write-Host "  http://localhost:8080/swagger/index.html" -ForegroundColor Cyan
Write-Host "`n"

Write-Success "Happy coding!"

