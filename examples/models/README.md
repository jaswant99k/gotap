# goTap Models Example

This example demonstrates how to use `goTap.Model` as a base for your database models.

## ⚠️ Important Note

This example uses SQLite which requires CGO. If you get a CGO error:

**Option 1: Enable CGO (Windows)**
```bash
$env:CGO_ENABLED=1
go build
```

**Option 2: Use PostgreSQL/MySQL instead**
Change the database driver in `main.go`:
```go
import "gorm.io/driver/postgres"  // or mysql

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
```

## Features

- **goTap.Model**: Base model with ID, CreatedAt, UpdatedAt, DeletedAt (soft delete support)
- **Relationships**: One-to-Many, Many-to-Many with junction tables
- **CRUD Operations**: Complete Create, Read, Update, Delete examples
- **Transactions**: Safe multi-table operations
- **Preloading**: Eager loading of relationships
- **Soft Deletes**: Records marked as deleted instead of removed

## Models Demonstrated

### User Model
```go
type User struct {
    goTap.Model
    Username     string       `gorm:"uniqueIndex;not null" json:"username"`
    Email        string       `gorm:"uniqueIndex;not null" json:"email"`
    PasswordHash string       `gorm:"not null" json:"-"`
    Role         string       `gorm:"default:'user'" json:"role"`
    IsActive     bool         `gorm:"default:true" json:"is_active"`
    Permissions  []Permission `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
}
```

### Product Model
```go
type Product struct {
    goTap.Model
    Name        string  `gorm:"not null" json:"name"`
    SKU         string  `gorm:"uniqueIndex;not null" json:"sku"`
    Description string  `json:"description"`
    Price       float64 `gorm:"not null" json:"price"`
    Stock       int     `gorm:"default:0" json:"stock"`
    Category    string  `gorm:"index" json:"category"`
    IsActive    bool    `gorm:"default:true" json:"is_active"`
}
```

### Order Model (with relationships)
```go
type Order struct {
    goTap.Model
    UserID      uint        `gorm:"not null;index" json:"user_id"`
    User        User        `gorm:"foreignKey:UserID" json:"user"`
    OrderItems  []OrderItem `gorm:"foreignKey:OrderID" json:"order_items"`
    TotalAmount float64     `gorm:"not null" json:"total_amount"`
    Status      string      `gorm:"default:'pending'" json:"status"`
}
```

## goTap.Model vs gorm.Model

### Using gorm.Model (standard)
```go
import "gorm.io/gorm"

type User struct {
    gorm.Model  // Embeds ID, CreatedAt, UpdatedAt, DeletedAt
    Username string
}
```

### Using goTap.Model (goTap-specific)
```go
import "github.com/jaswant99k/gotap"

type User struct {
    goTap.Model  // Same fields, goTap namespace
    Username string
}
```

## Benefits of goTap.Model

1. **Framework Integration**: Future goTap features can extend the model
2. **Swagger Support**: Pre-configured with Swagger annotations
3. **Consistent Namespace**: All framework types under `goTap.*`
4. **Example Tags**: Includes example values for API documentation
5. **Flexibility**: Works with all GORM features and relationships

## API Endpoints

### Users
- `POST /users` - Create user
- `GET /users/:id` - Get user with permissions
- `GET /users` - List all users
- `PUT /users/:id` - Update user
- `DELETE /users/:id` - Soft delete user

### Products
- `POST /products` - Create product
- `GET /products/:id` - Get product
- `GET /products?category=Electronics` - List products with filter

### Orders
- `POST /orders` - Create order (with transaction)
- `GET /orders/:id` - Get order with user and items

## Running the Example

```bash
cd examples/models
go run main.go
```

Server starts on http://localhost:8080

## Example Requests

### Create User
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "password_hash": "hashed_password",
    "role": "admin",
    "is_active": true
  }'
```

### Create Product
```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "sku": "LAP-001",
    "description": "High-performance laptop",
    "price": 999.99,
    "stock": 10,
    "category": "Electronics"
  }'
```

### Create Order
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "order_items": [
      {
        "product_id": 1,
        "quantity": 2,
        "price": 999.99
      }
    ],
    "total_amount": 1999.98,
    "status": "pending"
  }'
```

### Get User with Permissions
```bash
curl http://localhost:8080/users/1
```

### List Products by Category
```bash
curl http://localhost:8080/products?category=Electronics
```

## Key Features Demonstrated

### 1. Soft Deletes
When you delete a user, the record isn't removed from the database. Instead, `DeletedAt` is set:
```go
db.Delete(&user, id)  // Sets DeletedAt to current time
```

To permanently delete:
```go
db.Unscoped().Delete(&user, id)
```

### 2. Preloading Relationships
Efficiently load related data:
```go
// Load user with all permissions
db.Preload("Permissions").First(&user, id)

// Load order with user and all order items with products
db.Preload("User").Preload("OrderItems.Product").First(&order, id)
```

### 3. Transactions
Ensure data consistency:
```go
tx := db.Begin()
tx.Create(&order)
tx.Model(&Product{}).Update("stock", gorm.Expr("stock - ?", quantity))
tx.Commit()
```

### 4. Many-to-Many Relationships
Automatically manages junction tables:
```go
type User struct {
    goTap.Model
    Permissions []Permission `gorm:"many2many:user_permissions;"`
}
```

GORM creates `user_permissions` table with `user_id` and `permission_id`.

## Database Schema

The example creates these tables:
- `users` - User accounts
- `permissions` - Available permissions
- `user_permissions` - User-Permission junction table
- `products` - Product catalog
- `orders` - Customer orders
- `order_items` - Order line items

All tables include:
- `id` (primary key)
- `created_at` (timestamp)
- `updated_at` (timestamp)
- `deleted_at` (nullable timestamp for soft deletes)

## Migration

Auto-migrate on startup:
```go
db.AutoMigrate(&User{}, &Permission{}, &Product{}, &Order{}, &OrderItem{})
```

## Notes

- Both `goTap.Model` and `gorm.Model` work identically
- Use `goTap.Model` for consistency with the framework
- Use `gorm.Model` if you prefer standard GORM conventions
- All GORM features (hooks, callbacks, scopes) work with both
- `goTap.BaseModel` is an alias for backward compatibility

