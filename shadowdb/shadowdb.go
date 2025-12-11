// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package shadowdb

import (
	"database/sql"
	"errors"
	"sync"
	"time"
)

// Errors
var (
	ErrNoPrimaryDB        = errors.New("primary database not configured")
	ErrNoShadowDB         = errors.New("shadow database not configured")
	ErrBothDBsDown        = errors.New("both primary and shadow databases are down")
	ErrInvalidStrategy    = errors.New("invalid read/write strategy")
	ErrHealthCheckFailed  = errors.New("health check failed")
	ErrFailoverInProgress = errors.New("failover operation in progress")
)

// DBStatus represents database health status
type DBStatus string

const (
	StatusHealthy   DBStatus = "healthy"
	StatusDegraded  DBStatus = "degraded"
	StatusUnhealthy DBStatus = "unhealthy"
	StatusUnknown   DBStatus = "unknown"
)

// ReadStrategy defines how read operations are distributed
type ReadStrategy string

const (
	ReadPrimaryOnly  ReadStrategy = "primary-only"
	ReadShadowOnly   ReadStrategy = "shadow-only"
	ReadRoundRobin   ReadStrategy = "round-robin"
	ReadPrimaryFirst ReadStrategy = "primary-first"
	ReadShadowFirst  ReadStrategy = "shadow-first"
)

// WriteStrategy defines how write operations are handled
type WriteStrategy string

const (
	WritePrimaryOnly WriteStrategy = "primary-only"
	WriteBoth        WriteStrategy = "both"
	WriteShadowOnly  WriteStrategy = "shadow-only"
)

// Config holds Shadow DB configuration
type Config struct {
	// Primary database configuration
	Primary DBConfig

	// Shadow database configuration
	Shadow DBConfig

	// Read strategy
	ReadStrategy ReadStrategy

	// Write strategy
	WriteStrategy WriteStrategy

	// Auto failover enabled
	AutoFailover bool

	// Auto failback enabled (switch back to primary when it recovers)
	AutoFailback bool

	// Health check interval
	HealthCheckInterval time.Duration

	// Health check timeout
	HealthCheckTimeout time.Duration

	// Max failures before marking DB as unhealthy
	MaxFailures int

	// Failover callback
	OnFailover func(from, to string)

	// Failback callback
	OnFailback func()

	// Health status change callback
	OnHealthChange func(db string, oldStatus, newStatus DBStatus)
}

// DBConfig holds individual database configuration
type DBConfig struct {
	// DSN (Data Source Name)
	DSN string

	// Driver name (mysql, postgres, sqlite3, etc.)
	Driver string

	// Max open connections
	MaxOpenConns int

	// Max idle connections
	MaxIdleConns int

	// Connection max lifetime
	ConnMaxLifetime time.Duration

	// Connection max idle time
	ConnMaxIdleTime time.Duration
}

// ShadowDB manages dual database connections with failover
type ShadowDB struct {
	config Config

	primary       *sql.DB
	shadow        *sql.DB
	primaryHealth *HealthStatus
	shadowHealth  *HealthStatus

	mu              sync.RWMutex
	activePrimary   bool // true if primary is active, false if shadow is active
	failoverLock    sync.Mutex
	roundRobinCount uint64

	stopHealthCheck chan struct{}
	healthCheckWg   sync.WaitGroup
}

// HealthStatus tracks database health
type HealthStatus struct {
	mu               sync.RWMutex
	status           DBStatus
	lastCheck        time.Time
	lastSuccess      time.Time
	lastFailure      time.Time
	consecutiveFails int
	totalChecks      int
	totalFailures    int
}

// New creates a new ShadowDB instance
func New(config Config) (*ShadowDB, error) {
	if config.Primary.DSN == "" {
		return nil, ErrNoPrimaryDB
	}

	// Set defaults
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 10 * time.Second
	}
	if config.HealthCheckTimeout == 0 {
		config.HealthCheckTimeout = 5 * time.Second
	}
	if config.MaxFailures == 0 {
		config.MaxFailures = 3
	}
	if config.ReadStrategy == "" {
		config.ReadStrategy = ReadPrimaryFirst
	}
	if config.WriteStrategy == "" {
		config.WriteStrategy = WritePrimaryOnly
	}

	sdb := &ShadowDB{
		config:          config,
		activePrimary:   true,
		stopHealthCheck: make(chan struct{}),
		primaryHealth: &HealthStatus{
			status: StatusUnknown,
		},
		shadowHealth: &HealthStatus{
			status: StatusUnknown,
		},
	}

	// Connect to primary database
	if err := sdb.connectPrimary(); err != nil {
		return nil, err
	}

	// Connect to shadow database if configured
	if config.Shadow.DSN != "" {
		if err := sdb.connectShadow(); err != nil {
			// Log error but don't fail - shadow is optional
			if config.OnHealthChange != nil {
				config.OnHealthChange("shadow", StatusUnknown, StatusUnhealthy)
			}
		}
	}

	// Start health checks
	sdb.startHealthChecks()

	return sdb, nil
}

// connectPrimary establishes connection to primary database
func (sdb *ShadowDB) connectPrimary() error {
	db, err := sql.Open(sdb.config.Primary.Driver, sdb.config.Primary.DSN)
	if err != nil {
		return err
	}

	// Configure connection pool
	if sdb.config.Primary.MaxOpenConns > 0 {
		db.SetMaxOpenConns(sdb.config.Primary.MaxOpenConns)
	}
	if sdb.config.Primary.MaxIdleConns > 0 {
		db.SetMaxIdleConns(sdb.config.Primary.MaxIdleConns)
	}
	if sdb.config.Primary.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(sdb.config.Primary.ConnMaxLifetime)
	}
	if sdb.config.Primary.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(sdb.config.Primary.ConnMaxIdleTime)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	sdb.primary = db
	sdb.primaryHealth.updateStatus(StatusHealthy)
	return nil
}

// connectShadow establishes connection to shadow database
func (sdb *ShadowDB) connectShadow() error {
	db, err := sql.Open(sdb.config.Shadow.Driver, sdb.config.Shadow.DSN)
	if err != nil {
		return err
	}

	// Configure connection pool
	if sdb.config.Shadow.MaxOpenConns > 0 {
		db.SetMaxOpenConns(sdb.config.Shadow.MaxOpenConns)
	}
	if sdb.config.Shadow.MaxIdleConns > 0 {
		db.SetMaxIdleConns(sdb.config.Shadow.MaxIdleConns)
	}
	if sdb.config.Shadow.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(sdb.config.Shadow.ConnMaxLifetime)
	}
	if sdb.config.Shadow.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(sdb.config.Shadow.ConnMaxIdleTime)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	sdb.shadow = db
	sdb.shadowHealth.updateStatus(StatusHealthy)
	return nil
}

// Primary returns the primary database connection
func (sdb *ShadowDB) Primary() *sql.DB {
	sdb.mu.RLock()
	defer sdb.mu.RUnlock()
	return sdb.primary
}

// Shadow returns the shadow database connection
func (sdb *ShadowDB) Shadow() *sql.DB {
	sdb.mu.RLock()
	defer sdb.mu.RUnlock()
	return sdb.shadow
}

// Read returns a database connection for read operations based on strategy
func (sdb *ShadowDB) Read() (*sql.DB, error) {
	sdb.mu.RLock()
	defer sdb.mu.RUnlock()

	switch sdb.config.ReadStrategy {
	case ReadPrimaryOnly:
		if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
			return sdb.primary, nil
		}
		return nil, ErrNoPrimaryDB

	case ReadShadowOnly:
		if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			return sdb.shadow, nil
		}
		return nil, ErrNoShadowDB

	case ReadRoundRobin:
		sdb.roundRobinCount++
		if sdb.roundRobinCount%2 == 0 {
			if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
				return sdb.primary, nil
			}
			if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
				return sdb.shadow, nil
			}
		} else {
			if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
				return sdb.shadow, nil
			}
			if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
				return sdb.primary, nil
			}
		}
		return nil, ErrBothDBsDown

	case ReadPrimaryFirst:
		if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
			return sdb.primary, nil
		}
		if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			return sdb.shadow, nil
		}
		return nil, ErrBothDBsDown

	case ReadShadowFirst:
		if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			return sdb.shadow, nil
		}
		if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
			return sdb.primary, nil
		}
		return nil, ErrBothDBsDown

	default:
		return nil, ErrInvalidStrategy
	}
}

// Write returns a database connection for write operations based on strategy
func (sdb *ShadowDB) Write() (*sql.DB, error) {
	sdb.mu.RLock()
	defer sdb.mu.RUnlock()

	switch sdb.config.WriteStrategy {
	case WritePrimaryOnly:
		if sdb.activePrimary && sdb.primary != nil && sdb.primaryHealth.isHealthy() {
			return sdb.primary, nil
		}
		// Failover to shadow if primary is down and auto-failover is enabled
		if !sdb.activePrimary && sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			return sdb.shadow, nil
		}
		return nil, ErrNoPrimaryDB

	case WriteShadowOnly:
		if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			return sdb.shadow, nil
		}
		return nil, ErrNoShadowDB

	case WriteBoth:
		// For write-both, return primary (caller should handle dual writes)
		if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
			return sdb.primary, nil
		}
		if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			return sdb.shadow, nil
		}
		return nil, ErrBothDBsDown

	default:
		return nil, ErrInvalidStrategy
	}
}

// Close closes all database connections
func (sdb *ShadowDB) Close() error {
	// Stop health checks
	close(sdb.stopHealthCheck)
	sdb.healthCheckWg.Wait()

	var errs []error

	sdb.mu.Lock()
	defer sdb.mu.Unlock()

	if sdb.primary != nil {
		if err := sdb.primary.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if sdb.shadow != nil {
		if err := sdb.shadow.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// updateStatus updates health status
func (hs *HealthStatus) updateStatus(status DBStatus) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.status = status
	hs.lastCheck = time.Now()

	if status == StatusHealthy {
		hs.lastSuccess = time.Now()
		hs.consecutiveFails = 0
	} else {
		hs.lastFailure = time.Now()
		hs.consecutiveFails++
		hs.totalFailures++
	}
	hs.totalChecks++
}

// isHealthy checks if database is healthy
func (hs *HealthStatus) isHealthy() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.status == StatusHealthy
}

// GetStatus returns current status
func (hs *HealthStatus) GetStatus() DBStatus {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.status
}

// GetStats returns health statistics
func (hs *HealthStatus) GetStats() HealthStats {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	return HealthStats{
		Status:           hs.status,
		LastCheck:        hs.lastCheck,
		LastSuccess:      hs.lastSuccess,
		LastFailure:      hs.lastFailure,
		ConsecutiveFails: hs.consecutiveFails,
		TotalChecks:      hs.totalChecks,
		TotalFailures:    hs.totalFailures,
	}
}

// HealthStats holds health statistics
type HealthStats struct {
	Status           DBStatus
	LastCheck        time.Time
	LastSuccess      time.Time
	LastFailure      time.Time
	ConsecutiveFails int
	TotalChecks      int
	TotalFailures    int
}
