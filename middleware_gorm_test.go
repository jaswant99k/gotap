package goTap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test model
type TestProduct struct {
	ID        uint    `gorm:"primaryKey"`
	Name      string  `gorm:"not null"`
	Price     float64 `gorm:"type:decimal(10,2)"`
	Stock     int     `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func setupTestDB(t *testing.T) *gorm.DB {
	// Use MySQL for testing (user's target database)
	// For unit tests, we'll create a mock-like setup
	config := &DBConfig{
		Driver:       "mysql",
		DSN:          "test:test@tcp(localhost:3306)/gotap_test?charset=utf8mb4&parseTime=True&loc=Local",
		LogLevel:     logger.Silent,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	db, err := NewGormDB(config)
	if err != nil {
		// Skip tests if MySQL is not available
		t.Skipf("Skipping GORM tests: MySQL not available (%v)", err)
		return nil
	}

	// Auto migrate test model
	if err := db.AutoMigrate(&TestProduct{}); err != nil {
		t.Fatalf("Failed to migrate test model: %v", err)
	}

	// Clean up any existing test data
	db.Exec("DELETE FROM test_products")

	return db
}

func TestNewGormDB(t *testing.T) {
	tests := []struct {
		name    string
		config  *DBConfig
		wantErr bool
		skip    bool
	}{
		{
			name: "MySQL connection",
			config: &DBConfig{
				Driver:   "mysql",
				DSN:      "test:test@tcp(localhost:3306)/gotap_test?charset=utf8mb4&parseTime=True",
				LogLevel: logger.Silent,
			},
			wantErr: false,
			skip:    true, // Skip if MySQL not available
		},
		{
			name: "PostgreSQL connection",
			config: &DBConfig{
				Driver:   "postgres",
				DSN:      "host=localhost user=test password=test dbname=gotap_test port=5432 sslmode=disable",
				LogLevel: logger.Silent,
			},
			wantErr: false,
			skip:    true, // Skip if PostgreSQL not available
		},
		{
			name: "Unsupported driver",
			config: &DBConfig{
				Driver: "oracle",
				DSN:    "invalid",
			},
			wantErr: true,
			skip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Skipping database connection test (requires running database)")
			}
			db, err := NewGormDB(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGormDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && db == nil {
				t.Error("Expected database connection, got nil")
			}
		})
	}
}

func TestGormInject(t *testing.T) {
	db := setupTestDB(t)
	router := New()
	router.Use(GormInject(db))

	router.GET("/test", func(c *Context) {
		gormDB, ok := GetGorm(c)
		if !ok {
			c.String(500, "GORM not found")
			return
		}
		if gormDB == nil {
			c.String(500, "GORM is nil")
			return
		}
		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestGetGorm(t *testing.T) {
	db := setupTestDB(t)
	router := New()
	router.Use(GormInject(db))

	router.GET("/test", func(c *Context) {
		gormDB, ok := GetGorm(c)
		if !ok {
			t.Error("GetGorm() returned false, expected true")
		}
		if gormDB == nil {
			t.Error("GetGorm() returned nil database")
		}
		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

func TestMustGetGorm(t *testing.T) {
	db := setupTestDB(t)
	router := New()

	t.Run("with injection", func(t *testing.T) {
		router.Use(GormInject(db))
		router.GET("/with", func(c *Context) {
			defer func() {
				if r := recover(); r != nil {
					t.Error("MustGetGorm() panicked unexpectedly")
				}
			}()
			gormDB := MustGetGorm(c)
			if gormDB == nil {
				t.Error("MustGetGorm() returned nil")
			}
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/with", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	})

	t.Run("without injection", func(t *testing.T) {
		routerNoDb := New()
		routerNoDb.GET("/without", func(c *Context) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("MustGetGorm() should have panicked")
				}
			}()
			MustGetGorm(c)
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/without", nil)
		w := httptest.NewRecorder()
		routerNoDb.ServeHTTP(w, req)
	})
}

func TestGormHealthCheck(t *testing.T) {
	db := setupTestDB(t)
	router := New()
	router.Use(GormInject(db))
	router.GET("/health", GormHealthCheck())

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response contains expected fields
	body := w.Body.String()
	expectedFields := []string{"status", "database", "open_conns"}
	for _, field := range expectedFields {
		if !contains(body, field) {
			t.Errorf("Response missing field: %s", field)
		}
	}
}

func TestGormPagination(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		expectedPage int
		expectedSize int
	}{
		{
			name:         "default values",
			queryString:  "",
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name:         "custom values",
			queryString:  "page=2&page_size=50",
			expectedPage: 2,
			expectedSize: 50,
		},
		{
			name:         "invalid page",
			queryString:  "page=-1&page_size=10",
			expectedPage: 1,
			expectedSize: 10,
		},
		{
			name:         "page size too large",
			queryString:  "page=1&page_size=200",
			expectedPage: 1,
			expectedSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := New()
			router.GET("/test", func(c *Context) {
				pagination := NewGormPagination(c)

				if pagination.Page != tt.expectedPage {
					t.Errorf("Expected page %d, got %d", tt.expectedPage, pagination.Page)
				}
				if pagination.PageSize != tt.expectedSize {
					t.Errorf("Expected page size %d, got %d", tt.expectedSize, pagination.PageSize)
				}
			})

			req := httptest.NewRequest("GET", "/test?"+tt.queryString, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		})
	}
}

func TestGormPaginationOffset(t *testing.T) {
	tests := []struct {
		page           int
		pageSize       int
		expectedOffset int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{3, 50, 100},
		{5, 10, 40},
	}

	for _, tt := range tests {
		pagination := &GormPagination{
			Page:     tt.page,
			PageSize: tt.pageSize,
		}
		offset := pagination.Offset()
		if offset != tt.expectedOffset {
			t.Errorf("Page %d, Size %d: expected offset %d, got %d",
				tt.page, tt.pageSize, tt.expectedOffset, offset)
		}
	}
}

func TestAutoMigrate(t *testing.T) {
	db := setupTestDB(t)

	type TestModel struct {
		ID   uint
		Name string
	}

	err := AutoMigrate(db, &TestModel{})
	if err != nil {
		t.Errorf("AutoMigrate() error = %v", err)
	}

	// Verify table exists
	if !db.Migrator().HasTable(&TestModel{}) {
		t.Error("Table was not created")
	}
}

func TestGormCRUDOperations(t *testing.T) {
	db := setupTestDB(t)

	t.Run("Create", func(t *testing.T) {
		product := &TestProduct{
			Name:  "Test Product",
			Price: 99.99,
			Stock: 10,
		}

		err := GormCreate(db, product)
		if err != nil {
			t.Errorf("GormCreate() error = %v", err)
		}
		if product.ID == 0 {
			t.Error("Product ID not set after create")
		}
	})

	t.Run("Find", func(t *testing.T) {
		// Create test data
		products := []TestProduct{
			{Name: "Product 1", Price: 10.0, Stock: 5},
			{Name: "Product 2", Price: 20.0, Stock: 10},
			{Name: "Product 3", Price: 30.0, Stock: 15},
		}
		for _, p := range products {
			db.Create(&p)
		}

		var results []TestProduct
		pagination := &GormPagination{Page: 1, PageSize: 2}
		err := GormFind(db, &results, pagination)
		if err != nil {
			t.Errorf("GormFind() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		product := &TestProduct{
			Name:  "Find by ID Test",
			Price: 50.0,
			Stock: 5,
		}
		db.Create(product)

		var found TestProduct
		err := GormFindByID(db, &found, product.ID)
		if err != nil {
			t.Errorf("GormFindByID() error = %v", err)
		}
		if found.Name != product.Name {
			t.Errorf("Expected name %s, got %s", product.Name, found.Name)
		}
	})

	t.Run("Update", func(t *testing.T) {
		product := &TestProduct{
			Name:  "Update Test",
			Price: 100.0,
			Stock: 10,
		}
		db.Create(product)

		updates := map[string]interface{}{
			"price": 150.0,
			"stock": 20,
		}
		err := GormUpdate(db, product, updates)
		if err != nil {
			t.Errorf("GormUpdate() error = %v", err)
		}

		var updated TestProduct
		db.First(&updated, product.ID)
		if updated.Price != 150.0 {
			t.Errorf("Expected price 150.0, got %f", updated.Price)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		product := &TestProduct{
			Name:  "Delete Test",
			Price: 200.0,
			Stock: 5,
		}
		db.Create(product)

		err := GormDelete(db, product)
		if err != nil {
			t.Errorf("GormDelete() error = %v", err)
		}

		var deleted TestProduct
		result := db.First(&deleted, product.ID)
		if result.Error == nil {
			t.Error("Product should have been deleted")
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := GormCountRecords(db, &TestProduct{})
		if err != nil {
			t.Errorf("GormCountRecords() error = %v", err)
		}
		if count < 0 {
			t.Error("Count should be non-negative")
		}
	})

	t.Run("Exists", func(t *testing.T) {
		product := &TestProduct{
			Name:  "Exists Test",
			Price: 300.0,
			Stock: 5,
		}
		db.Create(product)

		exists, err := GormExists(db, &TestProduct{}, "name = ?", "Exists Test")
		if err != nil {
			t.Errorf("GormExists() error = %v", err)
		}
		if !exists {
			t.Error("Product should exist")
		}
	})
}

func TestGormBatchOperations(t *testing.T) {
	db := setupTestDB(t)

	t.Run("BatchInsert", func(t *testing.T) {
		products := []TestProduct{
			{Name: "Batch 1", Price: 10.0, Stock: 5},
			{Name: "Batch 2", Price: 20.0, Stock: 10},
			{Name: "Batch 3", Price: 30.0, Stock: 15},
		}

		err := GormBatchInsert(db, &products, 2)
		if err != nil {
			t.Errorf("GormBatchInsert() error = %v", err)
		}

		var count int64
		db.Model(&TestProduct{}).Where("name LIKE ?", "Batch%").Count(&count)
		if count != 3 {
			t.Errorf("Expected 3 products, got %d", count)
		}
	})

	t.Run("BatchUpdate", func(t *testing.T) {
		updates := map[string]interface{}{
			"stock": 100,
		}
		err := GormBatchUpdate(db, &TestProduct{}, updates)
		if err != nil {
			t.Errorf("GormBatchUpdate() error = %v", err)
		}
	})

	t.Run("BatchDelete", func(t *testing.T) {
		// Create some products first
		products := []TestProduct{
			{Name: "Delete 1", Price: 10.0, Stock: 5},
			{Name: "Delete 2", Price: 20.0, Stock: 10},
		}
		db.Create(&products)

		ids := []interface{}{products[0].ID, products[1].ID}
		err := GormBatchDelete(db, &TestProduct{}, ids)
		if err != nil {
			t.Errorf("GormBatchDelete() error = %v", err)
		}
	})
}

func TestWithTransaction(t *testing.T) {
	db := setupTestDB(t)

	t.Run("successful transaction", func(t *testing.T) {
		err := WithTransaction(db, func(tx *gorm.DB) error {
			product := &TestProduct{
				Name:  "Transaction Test",
				Price: 100.0,
				Stock: 10,
			}
			return tx.Create(product).Error
		})

		if err != nil {
			t.Errorf("Transaction failed: %v", err)
		}

		var count int64
		db.Model(&TestProduct{}).Where("name = ?", "Transaction Test").Count(&count)
		if count != 1 {
			t.Error("Product not created in transaction")
		}
	})

	t.Run("rollback on error", func(t *testing.T) {
		err := WithTransaction(db, func(tx *gorm.DB) error {
			product := &TestProduct{
				Name:  "Rollback Test",
				Price: 100.0,
				Stock: 10,
			}
			if err := tx.Create(product).Error; err != nil {
				return err
			}
			// Return error to trigger rollback
			return gorm.ErrInvalidTransaction
		})

		if err == nil {
			t.Error("Expected transaction to fail")
		}

		var count int64
		db.Model(&TestProduct{}).Where("name = ?", "Rollback Test").Count(&count)
		if count != 0 {
			t.Error("Product should not exist after rollback")
		}
	})
}

func TestGormCache(t *testing.T) {
	cache := NewGormCache()

	t.Run("Set and Get", func(t *testing.T) {
		cache.Set("key1", "value1")
		val, exists := cache.Get("key1")
		if !exists {
			t.Error("Key should exist in cache")
		}
		if val != "value1" {
			t.Errorf("Expected 'value1', got '%v'", val)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		cache.Set("key2", "value2")
		cache.Delete("key2")
		_, exists := cache.Get("key2")
		if exists {
			t.Error("Key should not exist after delete")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		cache.Set("key3", "value3")
		cache.Set("key4", "value4")
		cache.Clear()
		_, exists1 := cache.Get("key3")
		_, exists2 := cache.Get("key4")
		if exists1 || exists2 {
			t.Error("Cache should be empty after clear")
		}
	})
}

func TestGormTransaction(t *testing.T) {
	db := setupTestDB(t)
	router := New()
	router.Use(GormInject(db))

	t.Run("successful request", func(t *testing.T) {
		router.POST("/transaction", GormTransaction(), func(c *Context) {
			tx := MustGetGorm(c)
			product := &TestProduct{
				Name:  "Transaction Middleware Test",
				Price: 50.0,
				Stock: 5,
			}
			if err := tx.Create(product).Error; err != nil {
				c.JSON(500, H{"error": err.Error()})
				return
			}
			c.JSON(200, H{"id": product.ID})
		})

		req := httptest.NewRequest("POST", "/transaction", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestDefaultDBConfig(t *testing.T) {
	config := DefaultDBConfig()

	if config.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns 10, got %d", config.MaxIdleConns)
	}
	if config.MaxOpenConns != 100 {
		t.Errorf("Expected MaxOpenConns 100, got %d", config.MaxOpenConns)
	}
	if config.ConnMaxLifetime != time.Hour {
		t.Errorf("Expected ConnMaxLifetime 1h, got %v", config.ConnMaxLifetime)
	}
}
