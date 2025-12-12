# Modular Project Structure for goTap

This guide demonstrates how to organize your goTap project using a **feature-based modular structure** instead of the traditional layer-based approach.

## Why Modular Structure?

**Benefits:**
- ✅ **High Cohesion** - Related code stays together
- ✅ **Easy to Navigate** - Find all code for a feature in one place
- ✅ **Scalability** - Add new features without touching existing modules
- ✅ **Team Collaboration** - Different teams can own different modules
- ✅ **Reusability** - Modules can be extracted as packages
- ✅ **Clear Boundaries** - Enforces separation of concerns

## Project Structure

```
vervepos/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
│
├── modules/                  # Feature modules (addons)
│   ├── auth/                # Authentication & authorization
│   │   ├── models.go        # User, Role, Permission models
│   │   ├── handlers.go      # Login, Register, Logout handlers
│   │   ├── service.go       # Business logic (password hashing, token generation)
│   │   ├── repository.go    # Database operations
│   │   ├── middleware.go    # Auth-specific middleware
│   │   └── routes.go        # Route registration
│   │
│   ├── products/            # Product management
│   │   ├── models.go        # Product, Category models
│   │   ├── handlers.go      # CRUD handlers
│   │   ├── service.go       # Business logic (pricing, inventory)
│   │   ├── repository.go    # Database queries
│   │   └── routes.go        # Product routes
│   │
│   ├── orders/              # Order processing
│   │   ├── models.go        # Order, OrderItem, Payment models
│   │   ├── handlers.go      # Create order, update status handlers
│   │   ├── service.go       # Order validation, calculation
│   │   ├── repository.go    # Order database operations
│   │   ├── events.go        # Order events (created, completed, canceled)
│   │   └── routes.go        # Order routes
│   │
│   ├── customers/           # Customer management
│   │   ├── models.go        # Customer, Address models
│   │   ├── handlers.go      # Customer CRUD
│   │   ├── service.go       # Customer logic (loyalty points)
│   │   ├── repository.go    # Customer queries
│   │   └── routes.go        # Customer routes
│   │
│   └── inventory/           # Inventory management
│       ├── models.go        # Stock, StockMovement models
│       ├── handlers.go      # Stock handlers
│       ├── service.go       # Stock calculations
│       └── routes.go        # Inventory routes
│
├── shared/                  # Shared utilities across modules
│   ├── database/
│   │   └── connection.go    # DB connection setup
│   │
│   ├── middleware/          # Global middleware
│   │   ├── cors.go
│   │   ├── logger.go
│   │   └── recovery.go
│   │
│   ├── errors/              # Error handling
│   │   └── errors.go
│   │
│   └── utils/               # Common utilities
│       ├── validator.go
│       └── helpers.go
│
├── config/
│   ├── config.go            # Configuration loader
│   └── .env                 # Environment variables
│
├── docs/                    # API documentation
│   └── api.md
│
├── go.mod
├── go.sum
└── README.md
```

## Module Structure

Each module follows this pattern:

```
module_name/
├── models.go       # Data models (GORM structs)
├── handlers.go     # HTTP handlers (request/response)
├── service.go      # Business logic (pure functions)
├── repository.go   # Database operations (GORM queries)
├── routes.go       # Route registration
├── middleware.go   # Module-specific middleware (optional)
└── events.go       # Event handlers (optional)
```

### Responsibilities

- **models.go**: Define database models and DTOs
- **handlers.go**: Handle HTTP requests, validate input, call services
- **service.go**: Implement business logic, coordinate between repos
- **repository.go**: Execute database queries, handle data persistence
- **routes.go**: Register module routes with the router
- **middleware.go**: Module-specific middleware (if needed)
- **events.go**: Handle domain events (order created, payment received, etc.)

## Complete Example

### 1. Auth Module

**modules/auth/models.go**
```go
package auth

import (
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

type User struct {
    gorm.Model
    Username     string `gorm:"uniqueIndex;not null"`
    Email        string `gorm:"uniqueIndex;not null"`
    PasswordHash string `gorm:"not null"`
    Role         string `gorm:"default:'user'"`
    IsActive     bool   `gorm:"default:true"`
}

type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=3"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}

type AuthResponse struct {
    Token string      `json:"token"`
    User  interface{} `json:"user"`
}
```

**modules/auth/service.go**
```go
package auth

import (
    "errors"
    "time"
    "github.com/yourusername/goTap"
    "golang.org/x/crypto/bcrypt"
)

type Service struct {
    repo      *Repository
    jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
    return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) Register(req RegisterRequest) (*User, error) {
    // Hash password
    hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
    if err != nil {
        return nil, err
    }

    user := &User{
        Username:     req.Username,
        Email:        req.Email,
        PasswordHash: string(hash),
        Role:         "user",
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

    // Verify password
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        return "", nil, errors.New("invalid credentials")
    }

    // Generate JWT token
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
```

**modules/auth/repository.go**
```go
package auth

import (
    "gorm.io/gorm"
)

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
    if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *Repository) FindByID(id uint) (*User, error) {
    var user User
    if err := r.db.First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *Repository) Update(user *User) error {
    return r.db.Save(user).Error
}
```

**modules/auth/handlers.go**
```go
package auth

import (
    "github.com/yourusername/goTap"
)

type Handler struct {
    service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

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
        },
    })
}

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

    c.JSON(200, AuthResponse{
        Token: token,
        User: goTap.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
            "role":     user.Role,
        },
    })
}

func (h *Handler) GetProfile(c *goTap.Context) {
    claims, _ := goTap.GetJWTClaims(c)

    user, err := h.service.GetUserByID(uint(claims.UserID))
    if err != nil {
        c.JSON(404, goTap.H{"error": "User not found"})
        return
    }

    c.JSON(200, goTap.H{
        "id":         user.ID,
        "username":   user.Username,
        "email":      user.Email,
        "role":       user.Role,
        "created_at": user.CreatedAt,
    })
}
```

**modules/auth/routes.go**
```go
package auth

import (
    "github.com/yourusername/goTap"
)

// RegisterRoutes registers all auth module routes
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
}
```

### 2. Products Module

**modules/products/models.go**
```go
package products

import "gorm.io/gorm"

type Product struct {
    gorm.Model
    Name        string  `gorm:"not null" json:"name"`
    SKU         string  `gorm:"uniqueIndex;not null" json:"sku"`
    Description string  `json:"description"`
    Price       float64 `gorm:"not null" json:"price"`
    Stock       int     `gorm:"default:0" json:"stock"`
    CategoryID  uint    `json:"category_id"`
    IsActive    bool    `gorm:"default:true" json:"is_active"`
}

type CreateProductRequest struct {
    Name        string  `json:"name" binding:"required"`
    SKU         string  `json:"sku" binding:"required"`
    Description string  `json:"description"`
    Price       float64 `json:"price" binding:"required,gt=0"`
    Stock       int     `json:"stock" binding:"gte=0"`
    CategoryID  uint    `json:"category_id"`
}
```

**modules/products/service.go**
```go
package products

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
        CategoryID:  req.CategoryID,
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

func (s *Service) UpdateStock(id uint, quantity int) error {
    product, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }

    product.Stock += quantity
    return s.repo.Update(product)
}
```

**modules/products/repository.go**
```go
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
```

**modules/products/handlers.go**
```go
package products

import (
    "strconv"
    "github.com/yourusername/goTap"
)

type Handler struct {
    service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

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

func (h *Handler) GetProduct(c *goTap.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

    product, err := h.service.GetProductByID(uint(id))
    if err != nil {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }

    c.JSON(200, product)
}
```

**modules/products/routes.go**
```go
package products

import "github.com/yourusername/goTap"

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
```

### 3. Main Application

**cmd/server/main.go**
```go
package main

import (
    "log"
    "os"

    "yourapp/modules/auth"
    "yourapp/modules/products"
    "yourapp/modules/orders"
    "yourapp/shared/database"
    
    "github.com/yourusername/goTap"
)

func main() {
    // Load configuration
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        log.Fatal("JWT_SECRET environment variable is required")
    }

    // Initialize database
    db, err := database.Connect()
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // Auto-migrate all modules
    db.AutoMigrate(
        &auth.User{},
        &products.Product{},
        &orders.Order{},
        &orders.OrderItem{},
    )

    // Initialize goTap
    r := goTap.Default()
    r.Use(goTap.GormInject(db))

    // Initialize modules
    initAuthModule(r, db, jwtSecret)
    initProductsModule(r, db, jwtSecret)
    initOrdersModule(r, db, jwtSecret)

    // Start server
    log.Println("🚀 Server starting on :8080")
    r.Run(":8080")
}

func initAuthModule(r *goTap.Engine, db *gorm.DB, jwtSecret string) {
    repo := auth.NewRepository(db)
    service := auth.NewService(repo, jwtSecret)
    handler := auth.NewHandler(service)
    auth.RegisterRoutes(r, handler, jwtSecret)
}

func initProductsModule(r *goTap.Engine, db *gorm.DB, jwtSecret string) {
    repo := products.NewRepository(db)
    service := products.NewService(repo)
    handler := products.NewHandler(service)
    products.RegisterRoutes(r, handler, jwtSecret)
}

func initOrdersModule(r *goTap.Engine, db *gorm.DB, jwtSecret string) {
    repo := orders.NewRepository(db)
    service := orders.NewService(repo)
    handler := orders.NewHandler(service)
    orders.RegisterRoutes(r, handler, jwtSecret)
}
```

## Benefits of This Structure

1. **Isolation**: Each module is self-contained
2. **Testability**: Easy to test modules independently
3. **Reusability**: Modules can be extracted and reused
4. **Team Collaboration**: Teams can work on different modules
5. **Clear Dependencies**: Module dependencies are explicit
6. **Scalability**: Add new modules without modifying existing code

## When to Use Modular Structure

✅ **Use modular structure when:**
- Building medium to large applications
- Multiple teams working on the project
- You want to extract features as separate packages
- You need clear feature boundaries

❌ **Use layer-based structure when:**
- Building small applications (<10 endpoints)
- Single developer or small team
- Rapid prototyping
- Simple CRUD application

## Migration from Layer-Based to Modular

1. Create `modules/` directory
2. Create feature directories (`auth/`, `products/`, etc.)
3. Move related models, handlers, services into module directories
4. Create `routes.go` for each module
5. Update `main.go` to initialize modules
6. Update import paths

## Next Steps

- Add inter-module communication (events, message bus)
- Implement dependency injection container
- Add module-level configuration
- Create module templates/generators
- Add API documentation per module

---

**This modular structure scales better for real-world POS systems!** 🚀
