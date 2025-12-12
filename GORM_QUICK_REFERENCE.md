# GORM Quick Reference - goTap Framework

## Installation

```bash
go get gorm.io/gorm
go get gorm.io/driver/mysql
```

## Setup

```go
import (
    "github.com/yourusername/goTap"
    "gorm.io/gorm/logger"
)

// Configure
config := &goTap.DBConfig{
    Driver: "mysql",
    DSN: "user:pass@tcp(localhost:3306)/db?parseTime=True",
    MaxIdleConns: 10,
    MaxOpenConns: 100,
    ConnMaxLifetime: time.Hour,
    LogLevel: logger.Info,
}

// Connect
db, err := goTap.NewGormDB(config)

// Use in router
router.Use(goTap.GormInject(db))
```

## Models

```go
type Product struct {
    gorm.Model // Adds ID, CreatedAt, UpdatedAt, DeletedAt
    Name  string  `gorm:"not null" json:"name"`
    SKU   string  `gorm:"uniqueIndex" json:"sku"`
    Price float64 `gorm:"type:decimal(10,2)" json:"price"`
    Stock int     `gorm:"default:0" json:"stock"`
}

// Migrate
goTap.AutoMigrate(db, &Product{})
```

## CRUD Operations

```go
func handler(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    
    // CREATE
    product := &Product{Name: "Test", Price: 10.0}
    goTap.GormCreate(db, product)
    
    // READ ONE
    var product Product
    goTap.GormFindByID(db, &product, 1)
    
    // READ MANY
    var products []Product
    pagination := goTap.NewGormPagination(c)
    goTap.GormFind(db, &products, pagination)
    
    // UPDATE
    updates := map[string]interface{}{"price": 15.0}
    goTap.GormUpdate(db, &product, updates)
    
    // DELETE (soft)
    goTap.GormDelete(db, &product)
    
    // COUNT
    total, _ := goTap.GormCountRecords(db, &Product{})
    
    // EXISTS
    exists, _ := goTap.GormExists(db, &Product{}, "sku = ?", "ABC123")
}
```

## Pagination

```go
// Automatic from query params ?page=1&page_size=20
pagination := goTap.NewGormPagination(c)

var products []Product
pagination.Apply(db).Find(&products)

// Manual
pagination := &goTap.GormPagination{Page: 1, PageSize: 20}
offset := pagination.Offset() // 0
limit := pagination.Limit()   // 20
```

## Transactions

```go
err := goTap.WithTransaction(db, func(tx *gorm.DB) error {
    tx.Create(&order)
    tx.Create(&items)
    tx.Model(&product).UpdateColumn("stock", gorm.Expr("stock - ?", qty))
    return nil // Commit (or return error to rollback)
})
```

## Middleware

```go
// Inject DB into context
router.Use(goTap.GormInject(db))

// Health check endpoint
router.GET("/health", goTap.GormHealthCheck())

// Transaction middleware (auto-commit/rollback)
router.POST("/order", goTap.GormTransaction(), createOrder)

// Query logging
router.Use(goTap.GormLogger())
```

## Batch Operations

```go
// Batch Insert (100 records at a time)
products := []Product{{Name: "A"}, {Name: "B"}}
goTap.GormBatchInsert(db, &products, 100)

// Batch Update
updates := map[string]interface{}{"stock": gorm.Expr("stock + ?", 10)}
goTap.GormBatchUpdate(db, &Product{}, updates)

// Batch Delete
ids := []interface{}{1, 2, 3, 4, 5}
goTap.GormBatchDelete(db, &Product{}, ids)
```

## Associations

```go
type Order struct {
    gorm.Model
    CustomerID uint
    Customer   Customer `gorm:"foreignKey:CustomerID"`
    Items      []OrderItem `gorm:"foreignKey:OrderID"`
}

// Preload associations
var order Order
db.Preload("Customer").Preload("Items.Product").First(&order, 1)

// Create with associations
order := Order{
    CustomerID: 1,
    Items: []OrderItem{
        {ProductID: 1, Quantity: 2},
    },
}
db.Create(&order)
```

## Raw SQL

```go
// Execute
goTap.GormExecRaw(db, "UPDATE products SET stock = ? WHERE id = ?", 100, 1)

// Query
var products []Product
goTap.GormQueryRaw(db, &products, "SELECT * FROM products WHERE price > ?", 50.0)
```

## Filtering & Searching

```go
// Simple filter
db.Where("category = ?", "electronics").Find(&products)

// Multiple conditions
db.Where("price > ? AND stock > ?", 100, 0).Find(&products)

// LIKE search
db.Where("name LIKE ?", "%laptop%").Find(&products)

// IN clause
db.Where("id IN ?", []int{1, 2, 3}).Find(&products)

// Order
db.Order("price DESC").Find(&products)

// Limit
db.Limit(10).Find(&products)
```

## Validation

```go
type ProductInput struct {
    Name  string  `json:"name" binding:"required,min=3,max=255"`
    Price float64 `json:"price" binding:"required,gt=0"`
    Stock int     `json:"stock" binding:"required,gte=0"`
}

func create(c *goTap.Context) {
    var input ProductInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }
    // ...
}
```

## Soft Deletes

```go
// Soft delete (sets DeletedAt)
db.Delete(&product)

// Permanent delete
db.Unscoped().Delete(&product)

// Include soft deleted
db.Unscoped().Find(&products)

// Find only deleted
db.Unscoped().Where("deleted_at IS NOT NULL").Find(&products)
```

## Caching

```go
cache := goTap.NewGormCache()

func getProduct(c *goTap.Context) {
    id := c.Param("id")
    
    // Check cache
    if cached, exists := cache.Get("product:" + id); exists {
        c.JSON(200, cached)
        return
    }
    
    // Query DB
    var product Product
    db.First(&product, id)
    
    // Cache result
    cache.Set("product:"+id, product)
    c.JSON(200, product)
}
```

## Context Timeouts

```go
ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

db := goTap.MustGetGorm(c).WithContext(ctx)
db.Find(&products)
```

## Common Patterns

### List with Pagination & Filters
```go
func listProducts(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    pagination := goTap.NewGormPagination(c)
    
    category := c.Query("category")
    search := c.Query("search")
    
    query := db.Model(&Product{})
    if category != "" {
        query = query.Where("category = ?", category)
    }
    if search != "" {
        query = query.Where("name LIKE ?", "%"+search+"%")
    }
    
    var total int64
    query.Count(&total)
    
    var products []Product
    pagination.Apply(query).Find(&products)
    
    c.JSON(200, goTap.H{
        "data": products,
        "page": pagination.Page,
        "size": pagination.PageSize,
        "total": total,
    })
}
```

### Create with Transaction
```go
func createOrder(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    
    var input OrderInput
    c.ShouldBindJSON(&input)
    
    err := goTap.WithTransaction(db, func(tx *gorm.DB) error {
        order := &Order{CustomerID: input.CustomerID}
        if err := tx.Create(order).Error; err != nil {
            return err
        }
        
        for _, item := range input.Items {
            var product Product
            if err := goTap.GormFindByID(tx, &product, item.ProductID); err != nil {
                return err
            }
            
            if product.Stock < item.Quantity {
                return errors.New("insufficient stock")
            }
            
            orderItem := &OrderItem{
                OrderID: order.ID,
                ProductID: product.ID,
                Quantity: item.Quantity,
            }
            if err := tx.Create(orderItem).Error; err != nil {
                return err
            }
            
            tx.Model(&product).UpdateColumn("stock", gorm.Expr("stock - ?", item.Quantity))
        }
        
        return nil
    })
    
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, goTap.H{"message": "Order created"})
}
```

## DSN Formats

**MySQL:**
```
username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

**PostgreSQL:**
```
host=localhost user=username password=password dbname=database port=5432 sslmode=disable
```

**SQLite:**
```
path/to/database.db
```

## Error Handling

```go
if err := db.Create(&product).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        c.JSON(404, goTap.H{"error": "Not found"})
    } else if errors.Is(err, gorm.ErrDuplicatedKey) {
        c.JSON(400, goTap.H{"error": "Duplicate key"})
    } else {
        c.JSON(500, goTap.H{"error": err.Error()})
    }
    return
}
```

## Documentation

- Full Guide: `C:\goTap\examples\gorm\README.md`
- VervePOS Example: `C:\Users\verve\Music\vervepos\GORM_SETUP.md`
- GORM Docs: https://gorm.io/docs/
