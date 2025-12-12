# GORM Associations Guide - Like Django ORM

GORM supports all Django-style relationships: **One-to-One**, **One-to-Many (ForeignKey)**, **Many-to-Many**, and **Polymorphic**.

## Table of Contents
- [One-to-One](#one-to-one)
- [One-to-Many (ForeignKey)](#one-to-many-foreignkey)
- [Many-to-Many](#many-to-many)
- [Polymorphic Associations](#polymorphic-associations)
- [Self-Referencing](#self-referencing)
- [Custom Join Tables](#custom-join-tables)
- [Preloading (Select Related)](#preloading-select-related)
- [Complete POS Example](#complete-pos-example)

## One-to-One

### Django Style:
```python
class User(models.Model):
    name = models.CharField(max_length=100)

class Profile(models.Model):
    user = models.OneToOneField(User, on_delete=models.CASCADE)
    bio = models.TextField()
```

### GORM Equivalent:
```go
type User struct {
    gorm.Model
    Name    string
    Profile Profile `gorm:"foreignKey:UserID"`
}

type Profile struct {
    gorm.Model
    UserID uint   `gorm:"uniqueIndex"`
    Bio    string
}

// Usage
db.Preload("Profile").First(&user, 1)
```

## One-to-Many (ForeignKey)

### Django Style:
```python
class Customer(models.Model):
    name = models.CharField(max_length=100)

class Order(models.Model):
    customer = models.ForeignKey(Customer, on_delete=models.CASCADE)
    total = models.DecimalField(max_digits=10, decimal_places=2)
```

### GORM Equivalent:
```go
type Customer struct {
    gorm.Model
    Name   string
    Orders []Order `gorm:"foreignKey:CustomerID"`
}

type Order struct {
    gorm.Model
    CustomerID uint
    Total      float64 `gorm:"type:decimal(10,2)"`
}

// Usage
db.Preload("Orders").First(&customer, 1)
```

## Many-to-Many

### Django Style:
```python
class Student(models.Model):
    name = models.CharField(max_length=100)
    courses = models.ManyToManyField('Course', related_name='students')

class Course(models.Model):
    name = models.CharField(max_length=100)
```

### GORM Equivalent:
```go
type Student struct {
    gorm.Model
    Name    string
    Courses []Course `gorm:"many2many:student_courses;"`
}

type Course struct {
    gorm.Model
    Name     string
    Students []Student `gorm:"many2many:student_courses;"`
}

// Auto-creates join table: student_courses
// Columns: student_id, course_id

// Usage
// Add courses to student
student := Student{Name: "John"}
courses := []Course{{Name: "Math"}, {Name: "Science"}}
db.Create(&student)
db.Model(&student).Association("Courses").Append(&courses)

// Query with preload
db.Preload("Courses").Find(&students)
```

## Many-to-Many with Extra Fields

### Django Style:
```python
class Product(models.Model):
    name = models.CharField(max_length=100)

class Tag(models.Model):
    name = models.CharField(max_length=50)

class ProductTag(models.Model):
    product = models.ForeignKey(Product, on_delete=models.CASCADE)
    tag = models.ForeignKey(Tag, on_delete=models.CASCADE)
    created_by = models.CharField(max_length=100)
    created_at = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        unique_together = ('product', 'tag')
```

### GORM Equivalent:
```go
type Product struct {
    gorm.Model
    Name string
    Tags []Tag `gorm:"many2many:product_tags;"`
}

type Tag struct {
    gorm.Model
    Name     string
    Products []Product `gorm:"many2many:product_tags;"`
}

// Custom join table with extra fields
type ProductTag struct {
    ProductID  uint      `gorm:"primaryKey"`
    TagID      uint      `gorm:"primaryKey"`
    CreatedBy  string
    CreatedAt  time.Time
}

// Usage with custom join table
db.SetupJoinTable(&Product{}, "Tags", &ProductTag{})

// Add tags with extra fields
db.Exec(`
    INSERT INTO product_tags (product_id, tag_id, created_by, created_at)
    VALUES (?, ?, ?, ?)
`, productID, tagID, "admin", time.Now())
```

## Polymorphic Associations

### Django Style:
```python
from django.contrib.contenttypes.fields import GenericForeignKey
from django.contrib.contenttypes.models import ContentType

class Comment(models.Model):
    content = models.TextField()
    content_type = models.ForeignKey(ContentType, on_delete=models.CASCADE)
    object_id = models.PositiveIntegerField()
    content_object = GenericForeignKey('content_type', 'object_id')
```

### GORM Equivalent:
```go
type Comment struct {
    gorm.Model
    Content       string
    CommentableID uint
    CommentableType string
}

type Post struct {
    gorm.Model
    Title    string
    Comments []Comment `gorm:"polymorphic:Commentable;"`
}

type Video struct {
    gorm.Model
    URL      string
    Comments []Comment `gorm:"polymorphic:Commentable;"`
}

// Usage
post := Post{Title: "Hello"}
db.Create(&post)

comment := Comment{
    Content: "Great post!",
    CommentableID: post.ID,
    CommentableType: "posts",
}
db.Create(&comment)

// Query
db.Preload("Comments").Find(&posts)
```

## Self-Referencing (Like Categories)

### Django Style:
```python
class Category(models.Model):
    name = models.CharField(max_length=100)
    parent = models.ForeignKey('self', null=True, blank=True, on_delete=models.CASCADE)
```

### GORM Equivalent:
```go
type Category struct {
    gorm.Model
    Name       string
    ParentID   *uint
    Parent     *Category   `gorm:"foreignKey:ParentID"`
    Children   []Category  `gorm:"foreignKey:ParentID"`
}

// Usage
// Create parent category
parent := Category{Name: "Electronics"}
db.Create(&parent)

// Create child categories
child1 := Category{Name: "Laptops", ParentID: &parent.ID}
child2 := Category{Name: "Phones", ParentID: &parent.ID}
db.Create(&child1)
db.Create(&child2)

// Query with children
db.Preload("Children").Find(&categories)

// Query with parent
db.Preload("Parent").First(&category, child1.ID)
```

## Preloading (Select Related / Prefetch Related)

### Django Style:
```python
# select_related (for ForeignKey and OneToOne)
orders = Order.objects.select_related('customer')

# prefetch_related (for ManyToMany and reverse ForeignKey)
students = Student.objects.prefetch_related('courses')
```

### GORM Equivalent:
```go
// Simple preload (like select_related)
db.Preload("Customer").Find(&orders)

// Multiple preloads
db.Preload("Customer").Preload("Items").Find(&orders)

// Nested preload
db.Preload("Orders.Items.Product").Find(&customers)

// Conditional preload
db.Preload("Orders", "total > ?", 100).Find(&customers)

// Custom preload with conditions
db.Preload("Orders", func(db *gorm.DB) *gorm.DB {
    return db.Where("status = ?", "completed").Order("created_at DESC")
}).Find(&customers)

// Preload with select
db.Preload("Courses", func(db *gorm.DB) *gorm.DB {
    return db.Select("id", "name")
}).Find(&students)
```

## Complete POS Example with All Relationships

```go
package main

import (
    "time"
    "github.com/yourusername/goTap"
    "gorm.io/gorm"
)

// ============================================================================
// ONE-TO-ONE: User and Profile
// ============================================================================

type User struct {
    gorm.Model
    Username string
    Email    string  `gorm:"uniqueIndex"`
    Profile  Profile `gorm:"foreignKey:UserID"`
}

type Profile struct {
    gorm.Model
    UserID    uint `gorm:"uniqueIndex"`
    FirstName string
    LastName  string
    Phone     string
}

// ============================================================================
// ONE-TO-MANY: Customer and Transactions
// ============================================================================

type Customer struct {
    gorm.Model
    Name          string
    Email         string `gorm:"uniqueIndex"`
    Phone         string
    LoyaltyPoints int
    Transactions  []Transaction `gorm:"foreignKey:CustomerID"`
}

type Transaction struct {
    gorm.Model
    CustomerID    uint
    Customer      Customer            `gorm:"foreignKey:CustomerID"`
    Total         float64             `gorm:"type:decimal(10,2)"`
    Status        string              `gorm:"default:'pending'"`
    Items         []TransactionItem   `gorm:"foreignKey:TransactionID"`
    Payments      []Payment           `gorm:"foreignKey:TransactionID"`
}

// ============================================================================
// MANY-TO-MANY: Product and Categories
// ============================================================================

type Product struct {
    gorm.Model
    Name         string
    SKU          string `gorm:"uniqueIndex"`
    Price        float64 `gorm:"type:decimal(10,2)"`
    Stock        int
    Categories   []Category `gorm:"many2many:product_categories;"`
    Tags         []Tag      `gorm:"many2many:product_tags;"`
    Suppliers    []Supplier `gorm:"many2many:product_suppliers;"`
}

type Category struct {
    gorm.Model
    Name     string
    Products []Product `gorm:"many2many:product_categories;"`
    ParentID *uint
    Parent   *Category  `gorm:"foreignKey:ParentID"`
    Children []Category `gorm:"foreignKey:ParentID"`
}

type Tag struct {
    gorm.Model
    Name     string
    Products []Product `gorm:"many2many:product_tags;"`
}

// ============================================================================
// MANY-TO-MANY WITH EXTRA FIELDS: Product Suppliers
// ============================================================================

type Supplier struct {
    gorm.Model
    Name     string
    Email    string
    Phone    string
    Products []Product `gorm:"many2many:product_suppliers;"`
}

type ProductSupplier struct {
    ProductID  uint      `gorm:"primaryKey"`
    SupplierID uint      `gorm:"primaryKey"`
    Cost       float64   `gorm:"type:decimal(10,2)"`
    LeadTime   int       // days
    IsActive   bool      `gorm:"default:true"`
    CreatedAt  time.Time
}

// ============================================================================
// LINKING TABLES: Transaction Items
// ============================================================================

type TransactionItem struct {
    gorm.Model
    TransactionID uint
    Transaction   Transaction `gorm:"foreignKey:TransactionID"`
    ProductID     uint
    Product       Product `gorm:"foreignKey:ProductID"`
    Quantity      int
    Price         float64 `gorm:"type:decimal(10,2)"`
    Discount      float64 `gorm:"type:decimal(10,2);default:0"`
}

// ============================================================================
// POLYMORPHIC: Comments (can be on Products or Transactions)
// ============================================================================

type Comment struct {
    gorm.Model
    Content         string
    CommentableID   uint
    CommentableType string
    UserID          uint
    User            User
}

// ============================================================================
// ONE-TO-MANY: Transaction and Payments
// ============================================================================

type Payment struct {
    gorm.Model
    TransactionID uint
    Amount        float64 `gorm:"type:decimal(10,2)"`
    Method        string  // cash, card, etc.
    Reference     string
}

// ============================================================================
// USAGE EXAMPLES
// ============================================================================

func main() {
    db := setupDatabase()
    router := goTap.New()
    router.Use(goTap.GormInject(db))

    // Routes
    router.GET("/customers/:id", getCustomerWithTransactions)
    router.GET("/products/:id", getProductWithRelations)
    router.POST("/transactions", createTransaction)
    router.GET("/categories", getCategoriesWithChildren)

    router.Run(":8080")
}

// Get customer with all transactions and items
func getCustomerWithTransactions(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    id := c.Param("id")

    var customer Customer
    err := db.
        Preload("Transactions.Items.Product").
        Preload("Transactions.Payments").
        First(&customer, id).Error

    if err != nil {
        c.JSON(404, goTap.H{"error": "Customer not found"})
        return
    }

    c.JSON(200, customer)
}

// Get product with categories, tags, and suppliers
func getProductWithRelations(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    id := c.Param("id")

    var product Product
    err := db.
        Preload("Categories.Parent").
        Preload("Tags").
        Preload("Suppliers").
        First(&product, id).Error

    if err != nil {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }

    c.JSON(200, product)
}

// Create transaction with items and payments
func createTransaction(c *goTap.Context) {
    db := goTap.MustGetGorm(c)

    var input struct {
        CustomerID uint `json:"customer_id" binding:"required"`
        Items []struct {
            ProductID uint    `json:"product_id" binding:"required"`
            Quantity  int     `json:"quantity" binding:"required,gt=0"`
            Discount  float64 `json:"discount"`
        } `json:"items" binding:"required,min=1"`
        Payments []struct {
            Amount float64 `json:"amount" binding:"required,gt=0"`
            Method string  `json:"method" binding:"required"`
        } `json:"payments" binding:"required,min=1"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    err := goTap.WithTransaction(db, func(tx *gorm.DB) error {
        // Create transaction
        transaction := &Transaction{
            CustomerID: input.CustomerID,
            Status:     "pending",
        }
        if err := tx.Create(transaction).Error; err != nil {
            return err
        }

        var total float64

        // Create transaction items
        for _, item := range input.Items {
            var product Product
            if err := tx.First(&product, item.ProductID).Error; err != nil {
                return err
            }

            if product.Stock < item.Quantity {
                return gorm.ErrInvalidData
            }

            transItem := TransactionItem{
                TransactionID: transaction.ID,
                ProductID:     product.ID,
                Quantity:      item.Quantity,
                Price:         product.Price,
                Discount:      item.Discount,
            }
            if err := tx.Create(&transItem).Error; err != nil {
                return err
            }

            // Update stock
            tx.Model(&product).UpdateColumn("stock", gorm.Expr("stock - ?", item.Quantity))

            total += (product.Price * float64(item.Quantity)) - item.Discount
        }

        // Create payments
        var totalPaid float64
        for _, payment := range input.Payments {
            pay := Payment{
                TransactionID: transaction.ID,
                Amount:        payment.Amount,
                Method:        payment.Method,
            }
            if err := tx.Create(&pay).Error; err != nil {
                return err
            }
            totalPaid += payment.Amount
        }

        // Verify payment
        if totalPaid < total {
            return gorm.ErrInvalidData
        }

        // Update transaction total and status
        transaction.Total = total
        transaction.Status = "completed"
        if err := tx.Save(transaction).Error; err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(201, goTap.H{"message": "Transaction created successfully"})
}

// Get categories with parent-child hierarchy
func getCategoriesWithChildren(c *goTap.Context) {
    db := goTap.MustGetGorm(c)

    var categories []Category
    err := db.
        Preload("Children").
        Preload("Parent").
        Where("parent_id IS NULL").
        Find(&categories).Error

    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }

    c.JSON(200, categories)
}

// ============================================================================
// ASSOCIATION OPERATIONS
// ============================================================================

// Add categories to product
func addCategoriesToProduct(db *gorm.DB, productID uint, categoryIDs []uint) error {
    var product Product
    if err := db.First(&product, productID).Error; err != nil {
        return err
    }

    var categories []Category
    if err := db.Find(&categories, categoryIDs).Error; err != nil {
        return err
    }

    return db.Model(&product).Association("Categories").Append(&categories)
}

// Remove category from product
func removeCategoryFromProduct(db *gorm.DB, productID, categoryID uint) error {
    var product Product
    if err := db.First(&product, productID).Error; err != nil {
        return err
    }

    return db.Model(&product).Association("Categories").Delete(&Category{Model: gorm.Model{ID: categoryID}})
}

// Replace all categories
func replaceProductCategories(db *gorm.DB, productID uint, categoryIDs []uint) error {
    var product Product
    if err := db.First(&product, productID).Error; err != nil {
        return err
    }

    var categories []Category
    if err := db.Find(&categories, categoryIDs).Error; err != nil {
        return err
    }

    return db.Model(&product).Association("Categories").Replace(&categories)
}

// Clear all associations
func clearProductCategories(db *gorm.DB, productID uint) error {
    var product Product
    if err := db.First(&product, productID).Error; err != nil {
        return err
    }

    return db.Model(&product).Association("Categories").Clear()
}

// Count associations
func countProductCategories(db *gorm.DB, productID uint) (int64, error) {
    var product Product
    if err := db.First(&product, productID).Error; err != nil {
        return 0, err
    }

    return db.Model(&product).Association("Categories").Count(), nil
}

// ============================================================================
// SETUP
// ============================================================================

func setupDatabase() *gorm.DB {
    config := &goTap.DBConfig{
        Driver:          "mysql",
        DSN:             "user:pass@tcp(localhost:3306)/pos?parseTime=True",
        MaxIdleConns:    10,
        MaxOpenConns:    100,
        ConnMaxLifetime: time.Hour,
    }

    db, _ := goTap.NewGormDB(config)

    // Migrate all models
    goTap.AutoMigrate(db,
        &User{}, &Profile{},
        &Customer{}, &Transaction{}, &TransactionItem{}, &Payment{},
        &Product{}, &Category{}, &Tag{}, &Supplier{},
        &Comment{},
    )

    // Setup custom join table with extra fields
    db.SetupJoinTable(&Product{}, "Suppliers", &ProductSupplier{})

    return db
}
```

## Association API Reference

```go
// Append (add without removing existing)
db.Model(&user).Association("Languages").Append([]Language{languageZH, languageEN})

// Replace (remove old and add new)
db.Model(&user).Association("Languages").Replace([]Language{languageZH, languageEN})

// Delete (remove specific associations)
db.Model(&user).Association("Languages").Delete([]Language{languageZH, languageEN})

// Clear (remove all associations)
db.Model(&user).Association("Languages").Clear()

// Count
count := db.Model(&user).Association("Languages").Count()

// Find associations
var languages []Language
db.Model(&user).Association("Languages").Find(&languages)
```

## Comparison: Django vs GORM

| Feature | Django | GORM |
|---------|--------|------|
| **One-to-One** | `OneToOneField` | `gorm:"foreignKey:UserID"` with unique index |
| **One-to-Many** | `ForeignKey` | `gorm:"foreignKey:CustomerID"` |
| **Many-to-Many** | `ManyToManyField` | `gorm:"many2many:table_name;"` |
| **Polymorphic** | `GenericForeignKey` | `gorm:"polymorphic:Owner;"` |
| **Self-Reference** | `ForeignKey('self')` | `gorm:"foreignKey:ParentID"` |
| **Select Related** | `.select_related()` | `.Preload()` |
| **Prefetch Related** | `.prefetch_related()` | `.Preload()` |
| **Through Model** | `through='ModelName'` | Custom join table struct |
| **Related Name** | `related_name='orders'` | Reverse field definition |
| **On Delete** | `on_delete=CASCADE` | Database constraint |

## Best Practices

1. **Always use Preload**: Avoid N+1 queries
   ```go
   db.Preload("Orders.Items.Product").Find(&customers)
   ```

2. **Use Association API**: For many-to-many operations
   ```go
   db.Model(&product).Association("Categories").Append(&categories)
   ```

3. **Custom Join Tables**: When you need extra fields
   ```go
   db.SetupJoinTable(&Product{}, "Suppliers", &ProductSupplier{})
   ```

4. **Conditional Preloads**: For filtered associations
   ```go
   db.Preload("Orders", "status = ?", "completed").Find(&customers)
   ```

5. **Nested Preloads**: Use dot notation
   ```go
   db.Preload("Orders.Items.Product.Categories").Find(&customers)
   ```

Yes, **GORM fully supports Django-style relationships** including:
- ✅ ForeignKey (One-to-Many)
- ✅ OneToOneField (One-to-One)
- ✅ ManyToManyField (Many-to-Many)
- ✅ GenericForeignKey (Polymorphic)
- ✅ Self-referencing relationships
- ✅ Through models with extra fields
- ✅ Preloading (select_related/prefetch_related)

Just like Django! 🎉
