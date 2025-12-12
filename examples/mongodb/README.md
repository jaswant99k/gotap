# MongoDB Integration for goTap

Complete guide to using MongoDB with the goTap framework.

## Installation

```bash
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/bson
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/jaswant99k/gotap"
)

func main() {
    r := goTap.New()
    
    // Connect to MongoDB
    mongoClient, err := goTap.NewMongoClient("mongodb://localhost:27017", "pos_database")
    if err != nil {
        log.Fatal(err)
    }
    defer mongoClient.Close()
    
    // Inject MongoDB into context
    r.Use(goTap.MongoInject(mongoClient))
    
    // Health check endpoint
    r.GET("/health", goTap.MongoHealthCheck())
    
    r.GET("/products", getProducts)
    r.POST("/products", createProduct)
    
    r.Run(":8080")
}

func getProducts(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "products")
    
    products, err := repo.Find(c.Request.Context(), goTap.H{})
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, products)
}
```

## Core Features

### 1. Connection Management

```go
// Basic connection
mongoClient, err := goTap.NewMongoClient("mongodb://localhost:27017", "mydb")

// Connection with authentication
uri := "mongodb://user:password@localhost:27017"
mongoClient, err := goTap.NewMongoClient(uri, "mydb")

// MongoDB Atlas
uri := "mongodb+srv://user:pass@cluster.mongodb.net"
mongoClient, err := goTap.NewMongoClient(uri, "mydb")
```

### 2. CRUD Operations with MongoRepository

```go
func productHandlers(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "products")
    ctx := c.Request.Context()
    
    // INSERT
    product := goTap.H{
        "name": "Laptop",
        "price": 999.99,
        "category": "electronics",
        "stock": 50,
    }
    insertResult, err := repo.InsertOne(ctx, product)
    
    // INSERT MANY
    products := []interface{}{
        goTap.H{"name": "Mouse", "price": 29.99},
        goTap.H{"name": "Keyboard", "price": 79.99},
    }
    repo.InsertMany(ctx, products)
    
    // FIND ONE
    filter := goTap.H{"name": "Laptop"}
    product, err := repo.FindOne(ctx, filter)
    
    // FIND ALL
    allProducts, err := repo.Find(ctx, goTap.H{})
    
    // FIND BY ID
    objectID, _ := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
    product, err := repo.FindByID(ctx, objectID)
    
    // UPDATE
    update := goTap.H{"$set": goTap.H{"stock": 45}}
    result, err := repo.UpdateOne(ctx, filter, update)
    
    // UPDATE BY ID
    repo.UpdateByID(ctx, objectID, update)
    
    // DELETE
    repo.DeleteOne(ctx, filter)
    
    // DELETE BY ID
    repo.DeleteByID(ctx, objectID)
    
    // COUNT
    count, err := repo.CountDocuments(ctx, filter)
}
```

### 3. Transactions

```go
r.POST("/transfer", func(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    
    err := goTap.MongoTransaction(c, mongo, func(ctx context.Context) error {
        accounts := goTap.NewMongoRepository(mongo, "accounts")
        
        // Deduct from account A
        accounts.UpdateOne(ctx, 
            goTap.H{"account_id": "A"}, 
            goTap.H{"$inc": goTap.H{"balance": -100}})
        
        // Add to account B
        accounts.UpdateOne(ctx,
            goTap.H{"account_id": "B"},
            goTap.H{"$inc": goTap.H{"balance": 100}})
        
        return nil // Commit transaction
    })
    
    if err != nil {
        c.JSON(500, goTap.H{"error": "Transaction failed"})
        return
    }
    c.JSON(200, goTap.H{"message": "Transfer successful"})
})
```

### 4. Caching with MongoDB

```go
r.GET("/products/:id", func(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    cache := goTap.NewMongoCache(mongo, "cache", 3600) // 1 hour TTL
    
    productID := c.Param("id")
    cacheKey := "product:" + productID
    
    // Check cache
    cached, found := cache.Get(c.Request.Context(), cacheKey)
    if found {
        c.JSON(200, cached)
        return
    }
    
    // Fetch from database
    repo := goTap.NewMongoRepository(mongo, "products")
    product, _ := repo.FindByID(c.Request.Context(), objectID)
    
    // Store in cache
    cache.Set(c.Request.Context(), cacheKey, product)
    c.JSON(200, product)
})
```

### 5. Pagination

```go
r.GET("/products", func(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "products")
    
    // Auto-parse ?page=1&page_size=20
    pagination := goTap.NewMongoPagination(c)
    
    filter := goTap.H{"category": "electronics"}
    opts := options.Find().
        SetSkip(pagination.Skip()).
        SetLimit(pagination.Limit())
    
    products, _ := repo.Find(c.Request.Context(), filter, opts)
    
    c.JSON(200, goTap.H{
        "data": products,
        "page": pagination.Page,
        "page_size": pagination.PageSize,
    })
})
```

### 6. Text Search

```go
r.GET("/search", func(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    query := c.Query("q")
    
    // Create text index (do once during setup)
    goTap.CreateTextIndex(c.Request.Context(), mongo, "products", "name", "description")
    
    // Search
    results, err := goTap.MongoTextSearch(
        c.Request.Context(),
        mongo,
        "products",
        query,
        10, // limit
    )
    
    c.JSON(200, results)
})
```

### 7. Audit Logging

```go
r.Use(goTap.MongoAuditLog(mongoClient, "audit_logs"))

// Automatically logs:
// - Request method, path, body
// - Response status, body
// - User info, IP address
// - Timestamp, duration
```

### 8. Aggregation Pipeline

```go
r.GET("/sales/report", func(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "orders")
    
    pipeline := []interface{}{
        goTap.H{"$match": goTap.H{"status": "completed"}},
        goTap.H{"$group": goTap.H{
            "_id": "$product_id",
            "total_sales": goTap.H{"$sum": "$amount"},
            "count": goTap.H{"$sum": 1},
        }},
        goTap.H{"$sort": goTap.H{"total_sales": -1}},
        goTap.H{"$limit": 10},
    }
    
    results, err := repo.Aggregate(c.Request.Context(), pipeline)
    c.JSON(200, results)
})
```

## POS System Examples

### 1. Product Catalog with Variants

```go
type Product struct {
    ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
    Name        string              `bson:"name" json:"name"`
    SKU         string              `bson:"sku" json:"sku"`
    Category    string              `bson:"category" json:"category"`
    Price       float64             `bson:"price" json:"price"`
    Variants    []ProductVariant    `bson:"variants" json:"variants"`
    Stock       int                 `bson:"stock" json:"stock"`
    Attributes  map[string]interface{} `bson:"attributes" json:"attributes"`
    CreatedAt   time.Time           `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time           `bson:"updated_at" json:"updated_at"`
}

type ProductVariant struct {
    Size  string  `bson:"size" json:"size"`
    Color string  `bson:"color" json:"color"`
    Price float64 `bson:"price" json:"price"`
    Stock int     `bson:"stock" json:"stock"`
}

// Store product with variants
r.POST("/products", func(c *goTap.Context) {
    var product Product
    c.BindJSON(&product)
    
    product.CreatedAt = time.Now()
    product.UpdatedAt = time.Now()
    
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "products")
    
    result, err := repo.InsertOne(c.Request.Context(), product)
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, goTap.H{"id": result.InsertedID})
})
```

### 2. Customer Profiles with Purchase History

```go
type Customer struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"`
    Name          string            `bson:"name"`
    Email         string            `bson:"email"`
    Phone         string            `bson:"phone"`
    LoyaltyPoints int               `bson:"loyalty_points"`
    Preferences   map[string]interface{} `bson:"preferences"`
    PurchaseHistory []Purchase      `bson:"purchase_history"`
}

type Purchase struct {
    OrderID   string    `bson:"order_id"`
    Amount    float64   `bson:"amount"`
    Date      time.Time `bson:"date"`
    Products  []string  `bson:"products"`
}

// Get customer with purchase history
r.GET("/customers/:id/profile", func(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "customers")
    
    objectID, _ := primitive.ObjectIDFromHex(c.Param("id"))
    customer, err := repo.FindByID(c.Request.Context(), objectID)
    
    if err != nil {
        c.JSON(404, goTap.H{"error": "Customer not found"})
        return
    }
    
    c.JSON(200, customer)
})
```

### 3. Real-time Inventory Updates

```go
r.PUT("/inventory/update", func(c *goTap.Context) {
    type InventoryUpdate struct {
        ProductID string `json:"product_id"`
        Quantity  int    `json:"quantity"`
        Operation string `json:"operation"` // "add" or "subtract"
    }
    
    var update InventoryUpdate
    c.BindJSON(&update)
    
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "products")
    
    objectID, _ := primitive.ObjectIDFromHex(update.ProductID)
    
    var operation int
    if update.Operation == "add" {
        operation = update.Quantity
    } else {
        operation = -update.Quantity
    }
    
    filter := goTap.H{"_id": objectID}
    updateDoc := goTap.H{
        "$inc": goTap.H{"stock": operation},
        "$set": goTap.H{"updated_at": time.Now()},
    }
    
    result, err := repo.UpdateOne(c.Request.Context(), filter, updateDoc)
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, goTap.H{
        "matched": result.MatchedCount,
        "modified": result.ModifiedCount,
    })
})
```

### 4. Sales Analytics Dashboard

```go
r.GET("/analytics/sales", func(c *goTap.Context) {
    mongo := goTap.MustGetMongo(c)
    repo := goTap.NewMongoRepository(mongo, "orders")
    
    startDate := c.Query("start_date")
    endDate := c.Query("end_date")
    
    pipeline := []interface{}{
        goTap.H{"$match": goTap.H{
            "status": "completed",
            "created_at": goTap.H{
                "$gte": startDate,
                "$lte": endDate,
            },
        }},
        goTap.H{"$group": goTap.H{
            "_id": goTap.H{
                "year": goTap.H{"$year": "$created_at"},
                "month": goTap.H{"$month": "$created_at"},
                "day": goTap.H{"$dayOfMonth": "$created_at"},
            },
            "total_sales": goTap.H{"$sum": "$total"},
            "order_count": goTap.H{"$sum": 1},
            "avg_order": goTap.H{"$avg": "$total"},
        }},
        goTap.H{"$sort": goTap.H{"_id": 1}},
    }
    
    results, err := repo.Aggregate(c.Request.Context(), pipeline)
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, goTap.H{"analytics": results})
})
```

## Indexes for Performance

```go
// Create indexes during application startup
func setupIndexes(mongo *goTap.MongoClient) {
    ctx := context.Background()
    
    // Products
    products := mongo.Database.Collection("products")
    products.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{{Key: "sku", Value: 1}},
        Options: options.Index().SetUnique(true),
    })
    products.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{{Key: "category", Value: 1}},
    })
    
    // Customers
    customers := mongo.Database.Collection("customers")
    customers.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{{Key: "email", Value: 1}},
        Options: options.Index().SetUnique(true),
    })
    
    // Orders
    orders := mongo.Database.Collection("orders")
    orders.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{{Key: "customer_id", Value: 1}, {Key: "created_at", Value: -1}},
    })
}
```

## Testing

```go
func TestMongoRepository(t *testing.T) {
    // Set MONGODB_URI environment variable for testing
    // export MONGODB_URI="mongodb://localhost:27017"
    
    mongoClient := skipIfNoMongo(t)
    if mongoClient == nil {
        return // Test will be skipped
    }
    defer mongoClient.Close()
    
    repo := goTap.NewMongoRepository(mongoClient, "test_products")
    ctx := context.Background()
    
    // Insert test data
    product := goTap.H{"name": "Test Product", "price": 99.99}
    result, err := repo.InsertOne(ctx, product)
    if err != nil {
        t.Fatalf("Insert failed: %v", err)
    }
    
    // Verify
    found, err := repo.FindByID(ctx, result.InsertedID.(primitive.ObjectID))
    if err != nil {
        t.Fatalf("Find failed: %v", err)
    }
    
    if found["name"] != "Test Product" {
        t.Errorf("Expected 'Test Product', got %v", found["name"])
    }
}
```

## Best Practices

1. **Connection Pooling**: MongoDB driver handles connection pooling automatically
2. **Indexes**: Create indexes for frequently queried fields
3. **Pagination**: Always use pagination for large result sets
4. **Transactions**: Use transactions for multi-document operations
5. **Error Handling**: Always check errors from MongoDB operations
6. **Context**: Pass request context to MongoDB operations for cancellation
7. **Schema Validation**: Use MongoDB's built-in schema validation
8. **Backups**: Set up regular backups for production

## Performance Tips

- **Projection**: Only fetch needed fields using projection
- **Covered Queries**: Use indexes to cover queries (avoid collection scan)
- **Aggregation**: Use aggregation pipeline for complex queries
- **Batch Operations**: Use `InsertMany` instead of multiple `InsertOne`
- **Connection Limit**: Tune `maxPoolSize` for your workload
- **Read Preference**: Use read preference for read scaling

## Deployment

### MongoDB Atlas (Cloud)

```go
uri := "mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority"
mongoClient, err := goTap.NewMongoClient(uri, "production_db")
```

### Self-Hosted

```bash
# Docker
docker run -d -p 27017:27017 --name mongodb \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password \
  mongo:latest
```

## Troubleshooting

- **Connection Refused**: Check if MongoDB is running and accessible
- **Authentication Failed**: Verify username/password
- **Slow Queries**: Use `.explain()` to analyze query performance
- **Memory Issues**: Add indexes and use projection
- **Windows Testing**: MongoDB tests skip gracefully if server not available

## Learn More

- [MongoDB Go Driver Documentation](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo)
- [MongoDB Manual](https://docs.mongodb.com/manual/)
- [MongoDB Atlas](https://www.mongodb.com/cloud/atlas)
