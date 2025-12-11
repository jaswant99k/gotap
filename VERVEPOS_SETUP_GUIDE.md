# Using goTap Framework in VervePOS

## Database Support

**Yes! goTap supports both MySQL and PostgreSQL** through Go's standard `database/sql` package.

### Currently Supported (SQL Databases):
- ✅ **MySQL/MariaDB** - via `github.com/go-sql-driver/mysql`
- ✅ **PostgreSQL** - via `github.com/lib/pq`
- ✅ **SQLite** - via `github.com/mattn/go-sqlite3`
- ✅ **SQL Server** - via `github.com/denisenkom/go-mssqldb`
- ✅ **Any database/sql compatible driver**

### Future Support (Recommended for Modern POS):

#### 🚀 **NoSQL Databases** - Excellent for POS Systems
**Why Add NoSQL?**
- **High-speed caching**: Redis for real-time inventory, session management
- **Flexible schemas**: MongoDB for product catalogs, customer profiles
- **Horizontal scaling**: Handle Black Friday traffic spikes
- **Document storage**: Store receipts, invoices as JSON
- **Real-time analytics**: Fast aggregations for sales dashboards

**Recommended NoSQL Additions:**

1. **Redis** - Essential for POS
   - Use case: Session storage, cart management, real-time inventory cache
   - Driver: `github.com/redis/go-redis/v9`
   - Benefits: Sub-millisecond latency, pub/sub for real-time updates
   - Example: Cache hot products, track active terminals, rate limiting

2. **MongoDB** - Great for flexibility
   - Use case: Product catalogs with varying attributes, customer profiles, audit logs
   - Driver: `go.mongodb.org/mongo-driver/mongo`
   - Benefits: Flexible schema, powerful queries, change streams
   - Example: Store products with custom fields per category

3. **DynamoDB** (AWS) or **Cosmos DB** (Azure)
   - Use case: Global POS chains, serverless deployments
   - Benefits: Unlimited scale, multi-region replication
   - Example: Sync inventory across continents

#### 🔍 **Vector Databases** - Game-Changer for POS
**Why Add Vector Support?**
- **AI-powered search**: "Find red dress under $50" → semantic search
- **Product recommendations**: "Customers who bought X also bought..."
- **Visual search**: Upload photo to find similar products
- **Fraud detection**: Identify suspicious transaction patterns
- **Smart customer service**: ChatGPT-like help for staff

**Recommended Vector Database Additions:**

1. **pgvector** (PostgreSQL Extension) - Easiest start
   - Use case: Add AI features to existing PostgreSQL
   - Installation: `CREATE EXTENSION vector`
   - Benefits: No new database, SQL + vectors together
   - Example: Product embeddings for semantic search

2. **Pinecone** - Managed vector service
   - Use case: Production-ready AI search, recommendations
   - Driver: `github.com/pinecone-io/go-pinecone`
   - Benefits: Fully managed, scales automatically, low latency
   - Example: Real-time product recommendations

3. **Milvus** - Open-source vector DB
   - Use case: Self-hosted AI applications, large catalogs
   - Driver: `github.com/milvus-io/milvus-sdk-go/v2`
   - Benefits: Billion-scale vectors, GPU acceleration
   - Example: Visual product search across millions of items

4. **Weaviate** - GraphQL + vectors
   - Use case: Complex product relationships, multi-modal search
   - Benefits: Built-in ML models, hybrid search
   - Example: "Show me shoes similar to this image under $100"

5. **Qdrant** - Fast and accurate
   - Use case: Real-time recommendations, personalization
   - Driver: `github.com/qdrant/go-client`
   - Benefits: Fast filtering, payload indexing
   - Example: Personalized upselling based on purchase history

---

## Should You Add NoSQL + Vector Support?

### ✅ **YES - Strongly Recommended!**

**For POS Systems, here's the ideal setup:**

```
┌─────────────────────────────────────────────────────┐
│            goTap Framework Architecture              │
├─────────────────────────────────────────────────────┤
│                                                      │
│  SQL (MySQL/PostgreSQL)  ← Transactional data       │
│  └─ Orders, payments, inventory counts              │
│                                                      │
│  Redis (NoSQL)          ← Real-time caching         │
│  └─ Active carts, sessions, hot inventory           │
│                                                      │
│  MongoDB (NoSQL)        ← Flexible documents        │
│  └─ Product catalogs, customer profiles, logs       │
│                                                      │
│  pgvector/Pinecone      ← AI-powered features       │
│  └─ Product recommendations, semantic search        │
│                                                      │
└─────────────────────────────────────────────────────┘
```

### **Benefits for VervePOS:**

1. **Performance**: Redis caching → 100x faster inventory lookups
2. **Scalability**: NoSQL handles seasonal traffic spikes
3. **Flexibility**: MongoDB for products with varying attributes
4. **Modern Features**: Vector search for "smart product finder"
5. **Competitive Edge**: AI recommendations increase sales 15-30%
6. **Future-Proof**: Ready for ML/AI integration

### **Implementation Priority:**

**Phase 1: Essential (Implement Now)**
- ✅ Redis for caching and sessions
- ✅ SQL for transactions (already supported)

**Phase 2: Enhanced (3-6 months)**
- MongoDB for flexible product data
- pgvector for basic recommendations

**Phase 3: Advanced (6-12 months)**
- Pinecone for production AI features
- Real-time personalization

---

## Example: Hybrid Database Setup

### Multi-Database Context Injection

```go
package main

import (
    "context"
    "database/sql"
    "log"
    
    "github.com/redis/go-redis/v9"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    _ "github.com/go-sql-driver/mysql"
)

type DatabaseClients struct {
    MySQL   *sql.DB
    Redis   *redis.Client
    MongoDB *mongo.Client
}

func main() {
    // Initialize MySQL (transactional data)
    mysql, err := sql.Open("mysql", "user:pass@tcp(localhost:3306)/vervepos")
    if err != nil {
        log.Fatal("MySQL connection failed:", err)
    }
    defer mysql.Close()

    // Initialize Redis (caching, sessions)
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    defer rdb.Close()

    // Test Redis connection
    ctx := context.Background()
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Fatal("Redis connection failed:", err)
    }

    // Initialize MongoDB (product catalogs, logs)
    mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal("MongoDB connection failed:", err)
    }
    defer mongoClient.Disconnect(ctx)

    log.Println("✓ All databases connected successfully")

    // Create goTap router
    r := goTap.Default()

    // Inject all database clients into context
    dbs := &DatabaseClients{
        MySQL:   mysql,
        Redis:   rdb,
        MongoDB: mongoClient,
    }

    r.Use(func(c *goTap.Context) {
        c.Set("mysql", dbs.MySQL)
        c.Set("redis", dbs.Redis)
        c.Set("mongodb", dbs.MongoDB)
        c.Next()
    })

    // Setup routes with hybrid database access
    setupHybridRoutes(r)

    r.Run(":5066")
}

func setupHybridRoutes(r *goTap.Engine) {
    v1 := r.Group("/api/v1")
    {
        // Product endpoints with caching
        v1.GET("/products/:id", getProductWithCache)
        v1.POST("/products", createProductHybrid)
        
        // Smart search with vectors (future)
        v1.GET("/search", semanticSearch)
        
        // Real-time inventory with Redis
        v1.GET("/inventory/realtime", getRealtimeInventory)
    }
}

// Example: Get product with Redis caching
func getProductWithCache(c *goTap.Context) {
    ctx := context.Background()
    productID := c.Param("id")
    
    mysql := c.MustGet("mysql").(*sql.DB)
    rdb := c.MustGet("redis").(*redis.Client)
    
    // Try Redis cache first
    cacheKey := "product:" + productID
    cached, err := rdb.Get(ctx, cacheKey).Result()
    if err == nil {
        // Cache hit - return immediately
        c.Header("X-Cache", "HIT")
        c.Data(200, "application/json", []byte(cached))
        return
    }
    
    // Cache miss - query MySQL
    var product struct {
        ID    int     `json:"id"`
        Name  string  `json:"name"`
        Price float64 `json:"price"`
        Stock int     `json:"stock"`
    }
    
    err = mysql.QueryRow(
        "SELECT id, name, price, stock FROM products WHERE id = ?",
        productID,
    ).Scan(&product.ID, &product.Name, &product.Price, &product.Stock)
    
    if err == sql.ErrNoRows {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }
    if err != nil {
        c.JSON(500, goTap.H{"error": "Database error"})
        return
    }
    
    // Store in Redis cache (5 minutes TTL)
    productJSON, _ := json.Marshal(product)
    rdb.Set(ctx, cacheKey, productJSON, 5*time.Minute)
    
    c.Header("X-Cache", "MISS")
    c.JSON(200, product)
}

// Example: Create product in MongoDB (flexible schema)
func createProductHybrid(c *goTap.Context) {
    ctx := context.Background()
    mongodb := c.MustGet("mongodb").(*mongo.Client)
    
    var product map[string]interface{}
    if err := c.ShouldBindJSON(&product); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }
    
    // Store in MongoDB for flexible schema
    collection := mongodb.Database("vervepos").Collection("products")
    result, err := collection.InsertOne(ctx, product)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to create product"})
        return
    }
    
    c.JSON(201, goTap.H{
        "message": "Product created",
        "id":      result.InsertedID,
    })
}

// Example: Real-time inventory with Redis
func getRealtimeInventory(c *goTap.Context) {
    ctx := context.Background()
    rdb := c.MustGet("redis").(*redis.Client)
    
    // Get all inventory keys
    keys, err := rdb.Keys(ctx, "inventory:*").Result()
    if err != nil {
        c.JSON(500, goTap.H{"error": "Redis error"})
        return
    }
    
    inventory := make(map[string]string)
    for _, key := range keys {
        val, _ := rdb.Get(ctx, key).Result()
        inventory[key] = val
    }
    
    c.JSON(200, goTap.H{
        "inventory": inventory,
        "count":     len(inventory),
        "timestamp": time.Now(),
    })
}

// Example: Semantic search with vector database (future)
func semanticSearch(c *goTap.Context) {
    query := c.Query("q")
    
    // TODO: Implement with pgvector or Pinecone
    // 1. Convert query to embedding using OpenAI/local model
    // 2. Search vector database for similar products
    // 3. Return ranked results
    
    c.JSON(200, goTap.H{
        "message": "Semantic search coming soon",
        "query":   query,
    })
}
```

---

## Recommended Package Structure for Multi-Database

```go
// database/database.go
package database

import (
    "context"
    "database/sql"
    "github.com/redis/go-redis/v9"
    "go.mongodb.org/mongo-driver/mongo"
)

type Clients struct {
    SQL     *sql.DB
    Redis   *redis.Client
    MongoDB *mongo.Client
    // Future: Vector database client
}

func InitClients(sqlConn, redisAddr, mongoURI string) (*Clients, error) {
    // Initialize all database connections
    // Return unified client struct
}

// Middleware for goTap
func InjectDatabases(clients *Clients) goTap.HandlerFunc {
    return func(c *goTap.Context) {
        c.Set("dbs", clients)
        c.Next()
    }
}
```

---

## Quick Setup for VervePOS

### Step 1: Initialize Your Project

```powershell
cd C:\Users\verve\Music\vervepos
go mod init github.com/verve/vervepos

# Install goTap
go get github.com/yourusername/goTap

# Install database driver (choose one or both)
go get github.com/go-sql-driver/mysql      # For MySQL
go get github.com/lib/pq                    # For PostgreSQL
```

### Step 2: Create Main Application (`main.go`)

```go
package main

import (
    "database/sql"
    "log"
    "time"

    "github.com/yourusername/goTap"
    _ "github.com/go-sql-driver/mysql"  // MySQL driver
    // _ "github.com/lib/pq"             // PostgreSQL driver (if needed)
)

func main() {
    // Initialize database
    db, err := sql.Open("mysql", "username:password@tcp(localhost:3306)/vervepos?parseTime=true")
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    defer db.Close()

    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    // Test connection
    if err := db.Ping(); err != nil {
        log.Fatal("Database ping failed:", err)
    }
    log.Println("✓ Database connected successfully")

    // Create goTap router with middleware
    r := goTap.Default()

    // Add custom middleware to inject DB into context
    r.Use(func(c *goTap.Context) {
        c.Set("db", db)
        c.Next()
    })

    // Setup routes
    setupRoutes(r, db)

    // Start server
    log.Println("🚀 VervePOS API starting on :5066")
    r.Run(":5066")
}

func setupRoutes(r *goTap.Engine, db *sql.DB) {
    // Health check
    r.GET("/health", func(c *goTap.Context) {
        c.JSON(200, goTap.H{
            "status": "healthy",
            "service": "VervePOS API",
            "timestamp": time.Now().Format(time.RFC3339),
        })
    })

    // API v1 group
    v1 := r.Group("/api/v1")
    {
        // Products
        v1.GET("/products", getProducts)
        v1.GET("/products/:id", getProduct)
        v1.POST("/products", createProduct)
        v1.PUT("/products/:id", updateProduct)
        v1.DELETE("/products/:id", deleteProduct)

        // Transactions
        v1.POST("/transactions", createTransaction)
        v1.GET("/transactions/:id", getTransaction)
        v1.GET("/transactions", listTransactions)

        // Customers
        v1.GET("/customers", getCustomers)
        v1.POST("/customers", createCustomer)
    }
}

// Example handlers
func getProducts(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)
    
    rows, err := db.Query("SELECT id, name, price, stock FROM products WHERE active = 1")
    if err != nil {
        c.JSON(500, goTap.H{"error": "Database error"})
        return
    }
    defer rows.Close()

    products := []map[string]interface{}{}
    for rows.Next() {
        var id int
        var name string
        var price float64
        var stock int

        if err := rows.Scan(&id, &name, &price, &stock); err != nil {
            continue
        }

        products = append(products, map[string]interface{}{
            "id":    id,
            "name":  name,
            "price": price,
            "stock": stock,
        })
    }

    c.JSON(200, goTap.H{
        "products": products,
        "count":    len(products),
    })
}

func createProduct(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)

    var product struct {
        Name  string  `json:"name" binding:"required"`
        Price float64 `json:"price" binding:"required,min=0"`
        Stock int     `json:"stock" binding:"required,min=0"`
    }

    if err := c.ShouldBindJSON(&product); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    result, err := db.Exec(
        "INSERT INTO products (name, price, stock, active) VALUES (?, ?, ?, 1)",
        product.Name, product.Price, product.Stock,
    )
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to create product"})
        return
    }

    id, _ := result.LastInsertId()
    c.JSON(201, goTap.H{
        "message": "Product created successfully",
        "id":      id,
    })
}

func getProduct(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)
    id := c.Param("id")

    var product struct {
        ID    int     `json:"id"`
        Name  string  `json:"name"`
        Price float64 `json:"price"`
        Stock int     `json:"stock"`
    }

    err := db.QueryRow(
        "SELECT id, name, price, stock FROM products WHERE id = ? AND active = 1",
        id,
    ).Scan(&product.ID, &product.Name, &product.Price, &product.Stock)

    if err == sql.ErrNoRows {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }
    if err != nil {
        c.JSON(500, goTap.H{"error": "Database error"})
        return
    }

    c.JSON(200, product)
}

func updateProduct(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)
    id := c.Param("id")

    var updates map[string]interface{}
    if err := c.ShouldBindJSON(&updates); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    // Build dynamic UPDATE query
    query := "UPDATE products SET "
    args := []interface{}{}
    first := true

    if name, ok := updates["name"]; ok {
        if !first {
            query += ", "
        }
        query += "name = ?"
        args = append(args, name)
        first = false
    }
    if price, ok := updates["price"]; ok {
        if !first {
            query += ", "
        }
        query += "price = ?"
        args = append(args, price)
        first = false
    }
    if stock, ok := updates["stock"]; ok {
        if !first {
            query += ", "
        }
        query += "stock = ?"
        args = append(args, stock)
        first = false
    }

    query += " WHERE id = ?"
    args = append(args, id)

    result, err := db.Exec(query, args...)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Update failed"})
        return
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }

    c.JSON(200, goTap.H{"message": "Product updated successfully"})
}

func deleteProduct(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)
    id := c.Param("id")

    // Soft delete
    result, err := db.Exec("UPDATE products SET active = 0 WHERE id = ?", id)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Delete failed"})
        return
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        c.JSON(404, goTap.H{"error": "Product not found"})
        return
    }

    c.JSON(200, goTap.H{"message": "Product deleted successfully"})
}

func createTransaction(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)

    var txn struct {
        CustomerID int                      `json:"customer_id"`
        Items      []map[string]interface{} `json:"items" binding:"required"`
    }

    if err := c.ShouldBindJSON(&txn); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    // Start database transaction
    tx, err := db.Begin()
    if err != nil {
        c.JSON(500, goTap.H{"error": "Transaction failed"})
        return
    }
    defer tx.Rollback()

    // Insert transaction
    result, err := tx.Exec(
        "INSERT INTO transactions (customer_id, total, created_at) VALUES (?, 0, NOW())",
        txn.CustomerID,
    )
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to create transaction"})
        return
    }

    txnID, _ := result.LastInsertId()
    total := 0.0

    // Insert transaction items
    for _, item := range txn.Items {
        productID := int(item["product_id"].(float64))
        quantity := int(item["quantity"].(float64))

        // Get product price
        var price float64
        err := tx.QueryRow("SELECT price FROM products WHERE id = ?", productID).Scan(&price)
        if err != nil {
            c.JSON(400, goTap.H{"error": "Invalid product ID"})
            return
        }

        subtotal := price * float64(quantity)
        total += subtotal

        // Insert item
        _, err = tx.Exec(
            "INSERT INTO transaction_items (transaction_id, product_id, quantity, price, subtotal) VALUES (?, ?, ?, ?, ?)",
            txnID, productID, quantity, price, subtotal,
        )
        if err != nil {
            c.JSON(500, goTap.H{"error": "Failed to add item"})
            return
        }

        // Update stock
        _, err = tx.Exec("UPDATE products SET stock = stock - ? WHERE id = ?", quantity, productID)
        if err != nil {
            c.JSON(500, goTap.H{"error": "Failed to update stock"})
            return
        }
    }

    // Update transaction total
    _, err = tx.Exec("UPDATE transactions SET total = ? WHERE id = ?", total, txnID)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to update total"})
        return
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        c.JSON(500, goTap.H{"error": "Transaction commit failed"})
        return
    }

    c.JSON(201, goTap.H{
        "message":        "Transaction created successfully",
        "transaction_id": txnID,
        "total":          total,
    })
}

func getTransaction(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)
    id := c.Param("id")

    // Get transaction
    var txn struct {
        ID         int       `json:"id"`
        CustomerID int       `json:"customer_id"`
        Total      float64   `json:"total"`
        CreatedAt  time.Time `json:"created_at"`
    }

    err := db.QueryRow(
        "SELECT id, customer_id, total, created_at FROM transactions WHERE id = ?",
        id,
    ).Scan(&txn.ID, &txn.CustomerID, &txn.Total, &txn.CreatedAt)

    if err == sql.ErrNoRows {
        c.JSON(404, goTap.H{"error": "Transaction not found"})
        return
    }
    if err != nil {
        c.JSON(500, goTap.H{"error": "Database error"})
        return
    }

    // Get transaction items
    rows, err := db.Query(
        "SELECT product_id, quantity, price, subtotal FROM transaction_items WHERE transaction_id = ?",
        id,
    )
    if err != nil {
        c.JSON(500, goTap.H{"error": "Database error"})
        return
    }
    defer rows.Close()

    items := []map[string]interface{}{}
    for rows.Next() {
        var productID, quantity int
        var price, subtotal float64

        rows.Scan(&productID, &quantity, &price, &subtotal)
        items = append(items, map[string]interface{}{
            "product_id": productID,
            "quantity":   quantity,
            "price":      price,
            "subtotal":   subtotal,
        })
    }

    c.JSON(200, goTap.H{
        "id":          txn.ID,
        "customer_id": txn.CustomerID,
        "total":       txn.Total,
        "created_at":  txn.CreatedAt,
        "items":       items,
    })
}

func listTransactions(c *goTap.Context) {
    db := c.MustGet("db").(*sql.DB)

    // Get query parameters
    limit := c.DefaultQuery("limit", "20")
    offset := c.DefaultQuery("offset", "0")

    rows, err := db.Query(
        "SELECT id, customer_id, total, created_at FROM transactions ORDER BY created_at DESC LIMIT ? OFFSET ?",
        limit, offset,
    )
    if err != nil {
        c.JSON(500, goTap.H{"error": "Database error"})
        return
    }
    defer rows.Close()

    transactions := []map[string]interface{}{}
    for rows.Next() {
        var id, customerID int
        var total float64
        var createdAt time.Time

        rows.Scan(&id, &customerID, &total, &createdAt)
        transactions = append(transactions, map[string]interface{}{
            "id":          id,
            "customer_id": customerID,
            "total":       total,
            "created_at":  createdAt,
        })
    }

    c.JSON(200, goTap.H{
        "transactions": transactions,
        "count":        len(transactions),
    })
}

func getCustomers(c *goTap.Context) {
    // TODO: Implement customer listing
    c.JSON(200, goTap.H{"message": "Customers endpoint"})
}

func createCustomer(c *goTap.Context) {
    // TODO: Implement customer creation
    c.JSON(200, goTap.H{"message": "Create customer endpoint"})
}
```

### Step 3: PostgreSQL Configuration (Alternative)

If using PostgreSQL instead of MySQL:

```go
import (
    _ "github.com/lib/pq"  // PostgreSQL driver
)

func main() {
    // PostgreSQL connection string
    connStr := "host=localhost port=5432 user=postgres password=yourpass dbname=vervepos sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    
    // Rest of the code remains the same...
}
```

### Step 4: Database Schema (`schema.sql`)

```sql
-- Products table
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Customers table
CREATE TABLE customers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Transactions table
CREATE TABLE transactions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT,
    total DECIMAL(10, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

-- Transaction items table
CREATE TABLE transaction_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    transaction_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    subtotal DECIMAL(10, 2) NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Indexes for performance
CREATE INDEX idx_products_active ON products(active);
CREATE INDEX idx_transactions_customer ON transactions(customer_id);
CREATE INDEX idx_transactions_created ON transactions(created_at);
CREATE INDEX idx_items_transaction ON transaction_items(transaction_id);
```

### Step 5: Run Your Application

```powershell
cd C:\Users\verve\Music\vervepos
go run main.go
```

---

## Advanced Features

### 1. Add JWT Authentication

```go
import "github.com/yourusername/goTap"

func main() {
    r := goTap.Default()

    // Public routes
    r.POST("/login", login)

    // Protected routes
    authorized := r.Group("/api/v1")
    authorized.Use(goTap.JWTAuth("your-secret-key"))
    {
        authorized.GET("/products", getProducts)
        authorized.POST("/transactions", createTransaction)
    }

    r.Run(":5066")
}

func login(c *goTap.Context) {
    var creds struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&creds); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    // Verify credentials (check database)
    if creds.Username == "admin" && creds.Password == "password" {
        token, _ := goTap.GenerateJWT(map[string]interface{}{
            "user_id": 1,
            "role":    "admin",
        }, "your-secret-key", 24*3600) // 24 hours

        c.JSON(200, goTap.H{
            "token": token,
        })
        return
    }

    c.JSON(401, goTap.H{"error": "Invalid credentials"})
}
```

### 2. Add Rate Limiting

```go
r.Use(goTap.RateLimiter(100, time.Minute)) // 100 requests per minute
```

### 3. Add CORS

```go
r.Use(goTap.CORS(goTap.CORSConfig{
    AllowOrigins:     []string{"http://localhost:3000"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    AllowCredentials: true,
}))
```

### 4. Graceful Shutdown

```go
import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    r := goTap.Default()
    setupRoutes(r)

    // Start server in goroutine
    srv := r.RunServer(":5066")

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // Graceful shutdown with 5 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    log.Println("Server exited")
}
```

---

## Testing Your API

```powershell
# Health check
curl http://localhost:5066/health

# Get products
curl http://localhost:5066/api/v1/products

# Create product
curl -X POST http://localhost:5066/api/v1/products `
  -H "Content-Type: application/json" `
  -d '{\"name\":\"Coca Cola\",\"price\":2.50,\"stock\":100}'

# Get single product
curl http://localhost:5066/api/v1/products/1

# Create transaction
curl -X POST http://localhost:5066/api/v1/transactions `
  -H "Content-Type: application/json" `
  -d '{\"customer_id\":1,\"items\":[{\"product_id\":1,\"quantity\":2}]}'
```

---

## Project Structure (Recommended)

```
C:\Users\verve\Music\vervepos\
├── main.go                 # Application entry point
├── go.mod                  # Go module file
├── go.sum                  # Dependencies checksum
├── config/
│   └── config.go          # Configuration management
├── handlers/
│   ├── products.go        # Product handlers
│   ├── transactions.go    # Transaction handlers
│   └── customers.go       # Customer handlers
├── models/
│   ├── product.go         # Product model
│   ├── transaction.go     # Transaction model
│   └── customer.go        # Customer model
├── middleware/
│   └── auth.go            # Custom middleware
├── database/
│   ├── db.go              # Database connection
│   └── migrations/        # SQL migration files
└── schema.sql             # Database schema
```

---

---

## Installation Commands for All Database Types

### SQL Databases
```bash
go get github.com/go-sql-driver/mysql        # MySQL
go get github.com/lib/pq                      # PostgreSQL
go get github.com/mattn/go-sqlite3            # SQLite
```

### NoSQL Databases
```bash
go get github.com/redis/go-redis/v9           # Redis
go get go.mongodb.org/mongo-driver/mongo      # MongoDB
go get github.com/aws/aws-sdk-go-v2/service/dynamodb  # DynamoDB
```

### Vector Databases
```bash
go get github.com/pinecone-io/go-pinecone     # Pinecone
go get github.com/milvus-io/milvus-sdk-go/v2  # Milvus
go get github.com/qdrant/go-client            # Qdrant
# pgvector: Just enable extension in PostgreSQL
```

---

## Performance Comparison

### Query Latency (Typical POS Operations)

| Operation | SQL (MySQL) | Redis Cache | MongoDB | Vector Search |
|-----------|-------------|-------------|---------|---------------|
| Get Product | 5-15ms | 0.1-1ms | 2-5ms | 10-50ms |
| Search 1000 items | 50-200ms | N/A | 10-30ms | 20-100ms |
| Write Transaction | 10-50ms | 0.5-2ms | 5-15ms | 5-20ms |
| Complex Join | 100-500ms | N/A | 20-100ms | N/A |
| Semantic Search | N/A | N/A | N/A | 50-200ms |

**Key Takeaway**: Redis caching can make your POS **10-50x faster** for hot data!

---

## Real-World POS Use Cases

### Use Case 1: Fast Checkout (Redis + SQL)
```
Customer scans item → Check Redis cache for price → Instant response
Background: Update SQL for permanent record
Result: 100ms checkout time vs 500ms without cache
```

### Use Case 2: Smart Recommendations (Vector DB)
```
Customer adds item to cart → Find similar items using vectors
Show "Customers also bought..." with AI accuracy
Result: 15-30% increase in average transaction value
```

### Use Case 3: Flexible Product Catalog (MongoDB)
```
Clothing store: Store size, color, material per item
Electronics: Store specs like RAM, storage, warranty
No schema changes needed - just add new fields!
```

### Use Case 4: Real-Time Inventory (Redis Pub/Sub)
```
Item sold at Terminal 1 → Redis pub/sub → All terminals updated instantly
No more "item out of stock" surprises at checkout
Result: Better customer experience, reduced errors
```

---

## Framework Enhancement Recommendation

### New Middleware to Add to goTap:

1. **RedisCache Middleware** - Auto-cache GET requests
2. **MongoDBLogger Middleware** - Structured logging to MongoDB
3. **VectorSearch Middleware** - Add AI search to any endpoint
4. **HybridDB Middleware** - Automatic SQL+NoSQL coordination
5. **SessionRedis Middleware** - Redis-backed sessions

Would you like me to implement any of these middleware components?

---

## Migration Path

### Phase 1: Add Redis (Week 1)
- Install Redis locally/cloud
- Add caching middleware
- Cache hot products, inventory
- **Expected gain**: 50% faster response times

### Phase 2: Add MongoDB (Week 2-3)
- Set up MongoDB
- Migrate product catalog
- Store flexible customer data
- **Expected gain**: Flexible schema, easier development

### Phase 3: Add Vector DB (Week 4-6)
- Choose pgvector (easy) or Pinecone (powerful)
- Generate embeddings for products
- Implement semantic search
- **Expected gain**: AI-powered features, competitive advantage

---

## Next Steps

### For Current VervePOS Project:
1. **Install dependencies**: Run `go mod tidy`
2. **Setup SQL database**: Import `schema.sql` into MySQL/PostgreSQL
3. **Configure connection**: Update database credentials in `main.go`
4. **Run application**: `go run main.go`
5. **Test endpoints**: Use curl or Postman

### For Future Enhancements:
6. **Add Redis**: Install Redis, implement caching layer (1-2 days)
7. **Add MongoDB**: For flexible product catalog (3-5 days)
8. **Add Vector DB**: For AI recommendations (1-2 weeks)
9. **Performance testing**: Benchmark all database combinations
10. **Production deployment**: Use managed services (AWS RDS, Redis Cloud, MongoDB Atlas)

---

## Conclusion

**Should goTap support NoSQL and Vector databases?**

### 🎯 **ABSOLUTELY YES!**

**Why:**
- Modern POS systems need speed (Redis), flexibility (MongoDB), and AI (Vectors)
- Competitors are already using these technologies
- Implementation is straightforward with existing Go libraries
- Provides massive competitive advantage

**Priority:**
1. **Redis** - Essential (add immediately)
2. **MongoDB** - Very useful (add within 3 months)
3. **Vector DB** - Game-changer (add within 6 months)

The framework's current SQL support is excellent, but adding NoSQL and vector capabilities would make goTap a **truly modern, AI-ready POS framework** that can compete with any commercial solution.

Need help implementing any of these features? Let me know!
