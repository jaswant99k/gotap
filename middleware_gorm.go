package goTap

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConfig holds database configuration
type DBConfig struct {
	Driver          string        // "mysql", "postgres", "sqlite"
	DSN             string        // Data Source Name
	MaxIdleConns    int           // Maximum idle connections
	MaxOpenConns    int           // Maximum open connections
	ConnMaxLifetime time.Duration // Connection max lifetime
	LogLevel        logger.LogLevel
}

// DefaultDBConfig returns default database configuration
func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		LogLevel:        logger.Info,
	}
}

// NewGormDB creates a new GORM database connection
func NewGormDB(config *DBConfig) (*gorm.DB, error) {
	if config == nil {
		config = DefaultDBConfig()
	}

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(config.LogLevel)

	var dialector gorm.Dialector
	switch config.Driver {
	case "mysql":
		dialector = mysql.Open(config.DSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(config.DSN)
	case "sqlite":
		dialector = sqlite.Open(config.DSN)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", config.Driver)
	}

	// Open connection
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB for connection pooling
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("[GORM] Connected to %s database successfully", config.Driver)
	return db, nil
}

// DBConfigFromEnv loads database configuration from environment variables
func DBConfigFromEnv() *DBConfig {
	config := DefaultDBConfig()

	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		config.DSN = dsn
	}

	// Determine driver from DSN or explicit env var
	// Simple heuristic: check prefix or use DB_DRIVER env
	if driver := os.Getenv("DB_DRIVER"); driver != "" {
		config.Driver = driver
	} else {
		// Default to postgres if not specified but DSN looks like postgres
		// This is a fallback; explicit DB_DRIVER is better
		config.Driver = "postgres"
	}

	if maxIdle := os.Getenv("DB_MAX_IDLE_CONNS"); maxIdle != "" {
		if v, err := strconv.Atoi(maxIdle); err == nil {
			config.MaxIdleConns = v
		}
	}

	if maxOpen := os.Getenv("DB_MAX_OPEN_CONNS"); maxOpen != "" {
		if v, err := strconv.Atoi(maxOpen); err == nil {
			config.MaxOpenConns = v
		}
	}

	return config
}

// ConnectDB connects to the database using environment variables
func ConnectDB() (*gorm.DB, error) {
	config := DBConfigFromEnv()
	if config.DSN == "" {
		return nil, fmt.Errorf("DB_DSN environment variable is required")
	}
	return NewGormDB(config)
}

// GormInject injects GORM database instance into context
func GormInject(db *gorm.DB) HandlerFunc {
	return func(c *Context) {
		c.Set("gorm", db)
		c.Next()
	}
}

// GetGorm retrieves GORM database from context
func GetGorm(c *Context) (*gorm.DB, bool) {
	db, exists := c.Get("gorm")
	if !exists {
		return nil, false
	}
	gormDB, ok := db.(*gorm.DB)
	return gormDB, ok
}

// MustGetGorm retrieves GORM database from context or panics
func MustGetGorm(c *Context) *gorm.DB {
	db, ok := GetGorm(c)
	if !ok {
		panic("GORM database not found in context. Did you forget to use GormInject()?")
	}
	return db
}

// GormHealthCheck middleware for health check endpoint
func GormHealthCheck() HandlerFunc {
	return func(c *Context) {
		db, ok := GetGorm(c)
		if !ok {
			c.JSON(503, H{
				"status":   "unhealthy",
				"database": "not_configured",
			})
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(503, H{
				"status":   "unhealthy",
				"database": "error",
				"error":    err.Error(),
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(503, H{
				"status":   "unhealthy",
				"database": "connection_failed",
				"error":    err.Error(),
			})
			return
		}

		// Get connection stats
		stats := sqlDB.Stats()
		c.JSON(200, H{
			"status":         "healthy",
			"database":       "connected",
			"open_conns":     stats.OpenConnections,
			"in_use":         stats.InUse,
			"idle":           stats.Idle,
			"wait_count":     stats.WaitCount,
			"wait_duration":  stats.WaitDuration.String(),
			"max_idle_conns": stats.MaxIdleClosed,
			"max_lifetime":   stats.MaxLifetimeClosed,
		})
	}
}

// GormLogger middleware logs all database operations
func GormLogger() HandlerFunc {
	return func(c *Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)
		log.Printf("[GORM] %s %s | %v", c.Request.Method, path, duration)
	}
}

// GormTransaction wraps the handler in a database transaction
func GormTransaction() HandlerFunc {
	return func(c *Context) {
		db := MustGetGorm(c)

		// Begin transaction
		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(500, H{"error": "Failed to begin transaction"})
			c.Abort()
			return
		}

		// Replace db with transaction in context
		c.Set("gorm", tx)

		// Defer rollback in case of panic
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
				panic(r)
			}
		}()

		c.Next()

		// Check if there were any errors during request handling
		if len(c.Errors) > 0 {
			tx.Rollback()
			return
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			c.JSON(500, H{"error": "Failed to commit transaction"})
			return
		}
	}
}

// GormPagination helper for pagination
type GormPagination struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// NewGormPagination creates a new pagination instance from request
func NewGormPagination(c *Context) *GormPagination {
	pagination := &GormPagination{
		Page:     1,
		PageSize: 20,
	}
	c.ShouldBindQuery(pagination)

	// Validate
	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.PageSize < 1 {
		pagination.PageSize = 20
	}
	if pagination.PageSize > 100 {
		pagination.PageSize = 100
	}

	return pagination
}

// Offset calculates the offset for the query
func (p *GormPagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the page size
func (p *GormPagination) Limit() int {
	return p.PageSize
}

// Apply applies pagination to GORM query
func (p *GormPagination) Apply(db *gorm.DB) *gorm.DB {
	return db.Offset(p.Offset()).Limit(p.Limit())
}

// GormSoftDelete enables soft delete for all queries
type GormSoftDelete struct {
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// AutoMigrate runs auto migration for given models
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}
	log.Printf("[GORM] Auto migration completed for %d models", len(models))
	return nil
}

// GormCache is a simple caching helper for GORM queries
type GormCache struct {
	cache map[string]interface{}
}

// NewGormCache creates a new cache instance
func NewGormCache() *GormCache {
	return &GormCache{
		cache: make(map[string]interface{}),
	}
}

// Get retrieves value from cache
func (gc *GormCache) Get(key string) (interface{}, bool) {
	val, exists := gc.cache[key]
	return val, exists
}

// Set stores value in cache
func (gc *GormCache) Set(key string, value interface{}) {
	gc.cache[key] = value
}

// Delete removes value from cache
func (gc *GormCache) Delete(key string) {
	delete(gc.cache, key)
}

// Clear clears all cache
func (gc *GormCache) Clear() {
	gc.cache = make(map[string]interface{})
}

// GormWithContext returns GORM DB with request context
func GormWithContext(c *Context) *gorm.DB {
	db := MustGetGorm(c)
	return db.WithContext(c.Request.Context())
}

// GormBatchInsert performs batch insert operation
func GormBatchInsert(db *gorm.DB, records interface{}, batchSize int) error {
	return db.CreateInBatches(records, batchSize).Error
}

// GormBatchUpdate performs batch update operation
func GormBatchUpdate(db *gorm.DB, model interface{}, updates map[string]interface{}) error {
	return db.Model(model).Updates(updates).Error
}

// GormBatchDelete performs batch soft delete operation
func GormBatchDelete(db *gorm.DB, model interface{}, ids []interface{}) error {
	return db.Delete(model, ids).Error
}

// GormSearch performs full-text search (MySQL)
func GormSearch(db *gorm.DB, table, column, query string) *gorm.DB {
	return db.Table(table).Where(fmt.Sprintf("MATCH(%s) AGAINST(? IN BOOLEAN MODE)", column), query)
}

// GormExecRaw executes raw SQL query
func GormExecRaw(db *gorm.DB, sql string, values ...interface{}) error {
	return db.Exec(sql, values...).Error
}

// GormQueryRaw executes raw SQL query and scans results
func GormQueryRaw(db *gorm.DB, dest interface{}, sql string, values ...interface{}) error {
	return db.Raw(sql, values...).Scan(dest).Error
}

// WithTransaction helper function to run code in a transaction
func WithTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GormCountRecords counts total records for pagination
func GormCountRecords(db *gorm.DB, model interface{}, condition ...interface{}) (int64, error) {
	var count int64
	query := db.Model(model)
	if len(condition) > 0 {
		query = query.Where(condition[0], condition[1:]...)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GormFind helper for finding records with pagination
func GormFind(db *gorm.DB, dest interface{}, pagination *GormPagination, condition ...interface{}) error {
	query := db
	if len(condition) > 0 {
		query = query.Where(condition[0], condition[1:]...)
	}
	if pagination != nil {
		query = pagination.Apply(query)
	}
	return query.Find(dest).Error
}

// GormFindByID finds a record by ID
func GormFindByID(db *gorm.DB, dest interface{}, id interface{}) error {
	return db.First(dest, id).Error
}

// GormCreate creates a new record
func GormCreate(db *gorm.DB, value interface{}) error {
	return db.Create(value).Error
}

// GormUpdate updates a record
func GormUpdate(db *gorm.DB, model interface{}, updates interface{}) error {
	return db.Model(model).Updates(updates).Error
}

// GormDelete deletes a record (soft delete if model has DeletedAt)
func GormDelete(db *gorm.DB, value interface{}, conds ...interface{}) error {
	return db.Delete(value, conds...).Error
}

// GormExists checks if a record exists
func GormExists(db *gorm.DB, model interface{}, condition ...interface{}) (bool, error) {
	var count int64
	query := db.Model(model)
	if len(condition) > 0 {
		query = query.Where(condition[0], condition[1:]...)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
