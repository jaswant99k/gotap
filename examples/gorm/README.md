# goTap GORM Integration

Complete guide to using GORM ORM with goTap framework.

## Table of Contents
- [Overview](#overview)
- [Supported Databases](#supported-databases)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Database Models](#database-models)
- [CRUD Operations](#crud-operations)
- [Middleware](#middleware)
- [Transactions](#transactions)
- [Pagination](#pagination)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)

## Overview

goTap includes built-in GORM integration for elegant database operations with MySQL, PostgreSQL, and SQLite.

**Key Features:**
- Context injection middleware
- Connection pooling
- Transaction support
- Auto-migration
- Pagination helpers
- Query logging
- Health checks

## Supported Databases

### MySQL
```bash
go get gorm.io/driver/mysql
```

### PostgreSQL
```bash
go get gorm.io/driver/postgres
```

### SQLite
```bash
go get gorm.io/driver/sqlite
```

## Quick Start

### 1. Create Database Configuration

```go
package main

import (
    "log"
    "github.com/jaswant99k/gotap"
    "gorm.io/gorm/logger"
)

func main() {
    // Configure database
    dbConfig := &goTap.DBConfig{
        Driver:          "mysql",
        DSN:             "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
        MaxIdleConns:    10,
        MaxOpenConns:    100,
        ConnMaxLifetime: time.Hour,
        LogLevel:        logger.Info,
    }

    // Create database connection
    db, err := goTap.NewGormDB(dbConfig)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Connected to database!")
}
```

### 2. Use GORM with goTap Router

```go
func main() {
    // ... database setup from above

    // Create router
    router := goTap.New()

    // Inject GORM into context
    router.Use(goTap.GormInject(db))

    // Define routes
    router.GET("/products", getProducts)
    router.POST("/products", createProduct)

    // Start server
    router.Run(":8080")
}

// Handler using GORM
func getProducts(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    
    var products []Product
    if err := db.Find(&products).Error; err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(200, products)
}
```

## Configuration

### DSN Format

**MySQL:**
```
username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

**PostgreSQL:**
```
host=localhost user=username password=password dbname=database port=5432 sslmode=disable TimeZone=UTC
```

**SQLite:**
```
path/to/database.db
```

### Connection Pool Settings

```go
config := &goTap.DBConfig{
    MaxIdleConns:    10,   // Maximum idle connections
    MaxOpenConns:    100,  // Maximum open connections
    ConnMaxLifetime: time.Hour, // Connection lifetime
}
```

### Log Levels

```go
import "gorm.io/gorm/logger"

config.LogLevel = logger.Silent // No logs
config.LogLevel = logger.Error  // Errors only
config.LogLevel = logger.Warn   // Warnings and errors
config.LogLevel = logger.Info   // All queries
```

## Database Models

### Define Models

```go
import (
    "gorm.io/gorm"
    "time"
)

type Product struct {
    gorm.Model                    // Adds ID, CreatedAt, UpdatedAt, DeletedAt
    Name       string  `gorm:"not null;size:255"`
    SKU        string  `gorm:"uniqueIndex;not null;size:100"`
    Price      float64 `gorm:"type:decimal(10,2);not null"`
    Stock      int     `gorm:"default:0"`
    Category   string  `gorm:"index;size:100"`
}

type Customer struct {
    gorm.Model
    Name         string  `gorm:"not null;size:255"`
    Email        string  `gorm:"uniqueIndex;not null;size:255"`
    Phone        string  `gorm:"size:20"`
    LoyaltyPoints int    `gorm:"default:0"`
}

type Transaction struct {
    gorm.Model
    CustomerID    uint      `gorm:"index"`
    Customer      Customer  `gorm:"foreignKey:CustomerID"`
    Total         float64   `gorm:"type:decimal(10,2)"`
    PaymentMethod string    `gorm:"size:50"`
    Items         []TransactionItem `gorm:"foreignKey:TransactionID"`
}

type TransactionItem struct {
    ID            uint `gorm:"primaryKey"`
    TransactionID uint `gorm:"index"`
    ProductID     uint `gorm:"index"`
    Product       Product `gorm:"foreignKey:ProductID"`
    Quantity      int     `gorm:"not null"`
    Price         float64 `gorm:"type:decimal(10,2)"`
}
```

### Auto-Migration

```go
func main() {
    db, _ := goTap.NewGormDB(config)

    // Migrate all models
    err := goTap.AutoMigrate(db, 
        &Product{}, 
        &Customer{}, 
        &Transaction{}, 
        &TransactionItem{},
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## CRUD Operations

### Create

```go
func createProduct(c *goTap.Context) {
    db := goTap.MustGetGorm(c)

    var input Product
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    if err := goTap.GormCreate(db, &input); err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(201, input)
}
```

### Read (List with Pagination)

```go
func getProducts(c *goTap.Context) {
    db := goTap.MustGetGorm(c)

    // Get pagination params
    pagination := goTap.NewGormPagination(c)

    var products []Product
    if err := goTap.GormFind(db, &products, pagination); err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    // Get total count
    total, _ := goTap.GormCountRecords(db, &Product{})

    c.JSON(200, goTap.H{
        "data":      products,
        "page":      pagination.Page,
        "page_size": pagination.PageSize,
        "total":     total,
    })
}
```

### Read (Single Record)

```go
func getProduct(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    id := c.Param("id")

    var product Product
    if err := goTap.GormFindByID(db, &product, id); err != nil {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }

    c.JSON(200, product)
}
```

### Update

```go
func updateProduct(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    id := c.Param("id")

    var product Product
    if err := goTap.GormFindByID(db, &product, id); err != nil {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }

    var input Product
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    updates := map[string]interface{}{
        "name":     input.Name,
        "price":    input.Price,
        "stock":    input.Stock,
        "category": input.Category,
    }

    if err := goTap.GormUpdate(db, &product, updates); err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(200, product)
}
```

### Delete (Soft Delete)

```go
func deleteProduct(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    id := c.Param("id")

    var product Product
    if err := goTap.GormFindByID(db, &product, id); err != nil {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }

    if err := goTap.GormDelete(db, &product); err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(200, goTap.H{"message": "Product deleted"})
}
```

## Middleware

### GormInject - Context Injection

```go
router := goTap.New()
router.Use(goTap.GormInject(db))
```

### GormHealthCheck - Health Endpoint

```go
router.GET("/health", goTap.GormHealthCheck())
```

Response:
```json
{
    "status": "healthy",
    "database": "connected",
    "open_conns": 5,
    "in_use": 2,
    "idle": 3,
    "wait_count": 0,
    "wait_duration": "0s"
}
```

### GormLogger - Query Logging

```go
router.Use(goTap.GormLogger())
```

### GormTransaction - Auto Transaction

```go
router.POST("/orders", goTap.GormTransaction(), createOrder)

func createOrder(c *goTap.Context) {
    tx := goTap.MustGetGorm(c) // This is a transaction
    
    // All operations use transaction
    // Auto-commits if successful, rollbacks on error
}
```

## Transactions

### Manual Transaction

```go
func createTransaction(c *goTap.Context) {
    db := goTap.MustGetGorm(c)

    err := goTap.WithTransaction(db, func(tx *gorm.DB) error {
        // Create transaction
        transaction := &Transaction{
            CustomerID:    1,
            Total:         150.00,
            PaymentMethod: "cash",
        }
        if err := tx.Create(transaction).Error; err != nil {
            return err
        }

        // Create transaction items
        items := []TransactionItem{
            {TransactionID: transaction.ID, ProductID: 1, Quantity: 2, Price: 50.00},
            {TransactionID: transaction.ID, ProductID: 2, Quantity: 1, Price: 50.00},
        }
        if err := tx.Create(&items).Error; err != nil {
            return err
        }

        // Update product stock
        for _, item := range items {
            if err := tx.Model(&Product{}).Where("id = ?", item.ProductID).
                UpdateColumn("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
                return err
            }
        }

        return nil // Commit
    })

    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(201, goTap.H{"message": "Transaction created"})
}
```

## Pagination

### Basic Pagination

```go
func listItems(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    
    // Automatically parses ?page=1&page_size=20
    pagination := goTap.NewGormPagination(c)

    var items []Item
    pagination.Apply(db).Find(&items)

    c.JSON(200, goTap.H{
        "data": items,
        "page": pagination.Page,
        "size": pagination.PageSize,
    })
}
```

### Pagination with Filters

```go
func searchProducts(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    pagination := goTap.NewGormPagination(c)
    
    category := c.Query("category")

    var products []Product
    query := db.Model(&Product{})
    
    if category != "" {
        query = query.Where("category = ?", category)
    }

    // Get total before pagination
    var total int64
    query.Count(&total)

    // Apply pagination
    pagination.Apply(query).Find(&products)

    c.JSON(200, goTap.H{
        "data":  products,
        "page":  pagination.Page,
        "size":  pagination.PageSize,
        "total": total,
    })
}
```

## Advanced Features

### Batch Operations

```go
// Batch Insert
products := []Product{
    {Name: "Product 1", Price: 10.0, Stock: 100},
    {Name: "Product 2", Price: 20.0, Stock: 200},
    {Name: "Product 3", Price: 30.0, Stock: 300},
}
err := goTap.GormBatchInsert(db, &products, 100)

// Batch Update
updates := map[string]interface{}{
    "stock": gorm.Expr("stock + ?", 10),
}
err := goTap.GormBatchUpdate(db, &Product{}, updates)

// Batch Delete
ids := []interface{}{1, 2, 3, 4, 5}
err := goTap.GormBatchDelete(db, &Product{}, ids)
```

### Raw Queries

```go
// Execute raw SQL
err := goTap.GormExecRaw(db, 
    "UPDATE products SET stock = stock - ? WHERE id = ?", 
    10, 1,
)

// Query raw SQL
var results []Product
err := goTap.GormQueryRaw(db, &results,
    "SELECT * FROM products WHERE price > ? AND stock > ? LIMIT ?",
    50.0, 0, 10,
)
```

### Associations

```go
// Preload associations
var transaction Transaction
db.Preload("Customer").Preload("Items.Product").First(&transaction, 1)

// Create with associations
transaction := Transaction{
    CustomerID: 1,
    Total:      100.00,
    Items: []TransactionItem{
        {ProductID: 1, Quantity: 2, Price: 50.00},
    },
}
db.Create(&transaction)
```

### Caching

```go
cache := goTap.NewGormCache()

func getProduct(c *goTap.Context) {
    id := c.Param("id")
    
    // Check cache
    if cached, exists := cache.Get("product:" + id); exists {
        c.JSON(200, cached)
        return
    }

    // Query database
    db := goTap.MustGetGorm(c)
    var product Product
    if err := goTap.GormFindByID(db, &product, id); err != nil {
        c.JSON(404, goTap.H{"error": "Not found"})
        return
    }

    // Store in cache
    cache.Set("product:"+id, product)
    
    c.JSON(200, product)
}
```

## Best Practices

### 1. Use Transactions for Multiple Operations

```go
// Good: All operations in one transaction
err := goTap.WithTransaction(db, func(tx *gorm.DB) error {
    tx.Create(&order)
    tx.Create(&items)
    tx.Model(&product).Update("stock", newStock)
    return nil
})
```

### 2. Always Check Errors

```go
// Good
if err := db.Create(&product).Error; err != nil {
    return handleError(err)
}

// Bad
db.Create(&product)
```

### 3. Use Pagination for Large Datasets

```go
// Good
pagination := goTap.NewGormPagination(c)
goTap.GormFind(db, &products, pagination)

// Bad
db.Find(&products) // Loads all records
```

### 4. Use Indexes

```go
type Product struct {
    gorm.Model
    SKU      string `gorm:"uniqueIndex"` // Unique index
    Category string `gorm:"index"`       // Regular index
}
```

### 5. Use Soft Deletes

```go
type Product struct {
    gorm.Model // Includes DeletedAt for soft deletes
}

// Soft delete
db.Delete(&product) // Sets DeletedAt

// Permanent delete
db.Unscoped().Delete(&product)

// Include soft deleted
db.Unscoped().Find(&products)
```

### 6. Validate Input

```go
type ProductInput struct {
    Name  string  `json:"name" binding:"required,min=3,max=255"`
    Price float64 `json:"price" binding:"required,gt=0"`
    Stock int     `json:"stock" binding:"required,gte=0"`
}

func createProduct(c *goTap.Context) {
    var input ProductInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }
    // ...
}
```

### 7. Use Context Timeouts

```go
func queryWithTimeout(c *goTap.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
    defer cancel()

    db := goTap.MustGetGorm(c).WithContext(ctx)
    
    var products []Product
    if err := db.Find(&products).Error; err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(200, products)
}
```

## Complete Example

See `examples/gorm/main.go` for a complete working example of a REST API using goTap with GORM.

## Troubleshooting

### Connection Issues

```bash
# Test MySQL connection
mysql -u user -p -h localhost database

# Test PostgreSQL connection
psql -h localhost -U user -d database
```

### Migration Errors

```go
// Drop and recreate tables (development only)
db.Migrator().DropTable(&Product{}, &Customer{})
goTap.AutoMigrate(db, &Product{}, &Customer{})
```

### Performance Issues

```go
// Enable query logging
config.LogLevel = logger.Info

// Check slow queries
db.Debug().Find(&products)

// Optimize with indexes
type Product struct {
    Category string `gorm:"index"`
    SKU      string `gorm:"uniqueIndex"`
}
```

## Next Steps

- Check out the [Complete REST API Example](main.go)
- Learn about [Advanced Queries](https://gorm.io/docs/query.html)
- Read [GORM Documentation](https://gorm.io/docs/)
