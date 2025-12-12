package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jaswant99k/gotap/shadowdb"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/yourusername/goTap"
)

func main() {
	r := goTap.Default()

	// Configure Shadow DB
	sdb, err := shadowdb.New(shadowdb.Config{
		Primary: shadowdb.DBConfig{
			Driver:          "sqlite3",
			DSN:             "./primary.db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1 * time.Hour,
		},
		Shadow: shadowdb.DBConfig{
			Driver:          "sqlite3",
			DSN:             "./shadow.db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1 * time.Hour,
		},
		ReadStrategy:        shadowdb.ReadPrimaryFirst,
		WriteStrategy:       shadowdb.WritePrimaryOnly,
		AutoFailover:        true,
		AutoFailback:        true,
		HealthCheckInterval: 10 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		MaxFailures:         3,
		OnFailover: func(from, to string) {
			log.Printf("🔄 FAILOVER: Switched from %s to %s database", from, to)
		},
		OnFailback: func() {
			log.Printf("✅ FAILBACK: Switched back to primary database")
		},
		OnHealthChange: func(db string, oldStatus, newStatus shadowdb.DBStatus) {
			log.Printf("📊 HEALTH CHANGE: %s database status changed from %s to %s", db, oldStatus, newStatus)
		},
	})

	if err != nil {
		log.Fatalf("Failed to initialize Shadow DB: %v", err)
	}
	defer sdb.Close()

	// Initialize database schema
	initializeDatabase(sdb)

	// Add Shadow DB middleware
	r.Use(goTap.ShadowDBMiddleware(sdb))
	r.Use(goTap.DBHealthCheck())

	// Health check endpoint
	r.GET("/health", func(c *goTap.Context) {
		status := sdb.GetStatus()

		c.JSON(200, goTap.H{
			"status":         "ok",
			"active_db":      status.ActiveDB,
			"primary_health": status.PrimaryHealth,
			"shadow_health":  status.ShadowHealth,
			"read_strategy":  status.ReadStrategy,
			"write_strategy": status.WriteStrategy,
			"auto_failover":  status.AutoFailover,
			"auto_failback":  status.AutoFailback,
		})
	})

	// Database status endpoint
	r.GET("/db/status", func(c *goTap.Context) {
		status := sdb.GetStatus()
		c.JSON(200, status)
	})

	// Manual failover endpoint
	r.POST("/db/failover", func(c *goTap.Context) {
		if err := sdb.Failover(); err != nil {
			c.JSON(500, goTap.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(200, goTap.H{
			"message":   "Failover successful",
			"active_db": sdb.GetStatus().ActiveDB,
		})
	})

	// Manual failback endpoint
	r.POST("/db/failback", func(c *goTap.Context) {
		if err := sdb.Failback(); err != nil {
			c.JSON(500, goTap.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(200, goTap.H{
			"message":   "Failback successful",
			"active_db": sdb.GetStatus().ActiveDB,
		})
	})

	// CRUD operations with Shadow DB

	// Create transaction (uses write DB)
	r.POST("/transactions", func(c *goTap.Context) {
		if err := c.Request.ParseForm(); err != nil {
			c.JSON(400, goTap.H{"error": "Invalid request"})
			return
		}

		// Get Shadow DB from context
		shadowDB, _ := goTap.GetShadowDB(c)

		// Execute write operation
		result, err := shadowDB.ExecWrite(
			"INSERT INTO transactions (amount, description, created_at) VALUES (?, ?, ?)",
			c.PostForm("amount"),
			c.PostForm("description"),
			time.Now(),
		)

		if err != nil {
			c.JSON(500, goTap.H{"error": err.Error()})
			return
		}

		id, _ := result.LastInsertId()

		c.JSON(201, goTap.H{
			"id":      id,
			"message": "Transaction created",
			"db_used": shadowDB.GetStatus().ActiveDB,
		})
	})

	// List transactions (uses read DB)
	r.GET("/transactions", func(c *goTap.Context) {
		shadowDB, _ := goTap.GetShadowDB(c)

		rows, err := shadowDB.QueryRead("SELECT id, amount, description, created_at FROM transactions ORDER BY id DESC LIMIT 10")
		if err != nil {
			c.JSON(500, goTap.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var transactions []goTap.H
		for rows.Next() {
			var id int64
			var amount float64
			var description string
			var createdAt time.Time

			if err := rows.Scan(&id, &amount, &description, &createdAt); err != nil {
				continue
			}

			transactions = append(transactions, goTap.H{
				"id":          id,
				"amount":      amount,
				"description": description,
				"created_at":  createdAt,
			})
		}

		c.JSON(200, goTap.H{
			"transactions": transactions,
			"count":        len(transactions),
			"db_used":      shadowDB.GetStatus().ActiveDB,
		})
	})

	// Get single transaction
	r.GET("/transactions/:id", func(c *goTap.Context) {
		id := c.Param("id")
		shadowDB, _ := goTap.GetShadowDB(c)

		var amount float64
		var description string
		var createdAt time.Time

		row := shadowDB.QueryRowRead("SELECT amount, description, created_at FROM transactions WHERE id = ?", id)
		if err := row.Scan(&amount, &description, &createdAt); err != nil {
			c.JSON(404, goTap.H{"error": "Transaction not found"})
			return
		}

		c.JSON(200, goTap.H{
			"id":          id,
			"amount":      amount,
			"description": description,
			"created_at":  createdAt,
		})
	})

	// Transaction example using BeginTx
	r.POST("/transactions/batch", func(c *goTap.Context) {
		shadowDB, _ := goTap.GetShadowDB(c)

		tx, err := shadowDB.BeginTx()
		if err != nil {
			c.JSON(500, goTap.H{"error": err.Error()})
			return
		}

		// Insert multiple transactions
		for i := 1; i <= 3; i++ {
			_, err := tx.Exec(
				"INSERT INTO transactions (amount, description, created_at) VALUES (?, ?, ?)",
				float64(i*100),
				fmt.Sprintf("Batch transaction %d", i),
				time.Now(),
			)
			if err != nil {
				tx.Rollback()
				c.JSON(500, goTap.H{"error": err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(500, goTap.H{"error": err.Error()})
			return
		}

		c.JSON(201, goTap.H{
			"message": "Batch transactions created",
			"count":   3,
		})
	})

	log.Println("🚀 Server starting on :5066")
	log.Println("📊 Shadow DB configured with auto-failover")
	log.Println("Try these endpoints:")
	log.Println("  GET  /health - System health")
	log.Println("  GET  /db/status - Database status")
	log.Println("  POST /db/failover - Manual failover")
	log.Println("  POST /db/failback - Manual failback")
	log.Println("  POST /transactions - Create transaction")
	log.Println("  GET  /transactions - List transactions")

	r.Run(":5066")
}

func initializeDatabase(sdb *shadowdb.ShadowDB) {
	schema := `
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		amount REAL NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL
	);
	`

	// Initialize primary database
	if primary := sdb.Primary(); primary != nil {
		if _, err := primary.Exec(schema); err != nil {
			log.Printf("Warning: Failed to initialize primary DB: %v", err)
		} else {
			log.Println("✅ Primary database initialized")
		}
	}

	// Initialize shadow database
	if shadow := sdb.Shadow(); shadow != nil {
		if _, err := shadow.Exec(schema); err != nil {
			log.Printf("Warning: Failed to initialize shadow DB: %v", err)
		} else {
			log.Println("✅ Shadow database initialized")
		}
	}
}
