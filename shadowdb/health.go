// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package shadowdb

import (
	"context"
	"database/sql"
	"time"
)

// startHealthChecks starts periodic health checks
func (sdb *ShadowDB) startHealthChecks() {
	sdb.healthCheckWg.Add(1)
	go func() {
		defer sdb.healthCheckWg.Done()

		ticker := time.NewTicker(sdb.config.HealthCheckInterval)
		defer ticker.Stop()

		// Initial health check
		sdb.performHealthCheck()

		for {
			select {
			case <-ticker.C:
				sdb.performHealthCheck()
			case <-sdb.stopHealthCheck:
				return
			}
		}
	}()
}

// performHealthCheck checks health of both databases
func (sdb *ShadowDB) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), sdb.config.HealthCheckTimeout)
	defer cancel()

	// Check primary
	primaryHealthy := sdb.checkDatabaseHealth(ctx, sdb.primary, "primary")
	oldPrimaryStatus := sdb.primaryHealth.GetStatus()

	if primaryHealthy {
		sdb.primaryHealth.updateStatus(StatusHealthy)
	} else {
		if sdb.primaryHealth.consecutiveFails >= sdb.config.MaxFailures {
			sdb.primaryHealth.updateStatus(StatusUnhealthy)
		} else {
			sdb.primaryHealth.updateStatus(StatusDegraded)
		}
	}

	newPrimaryStatus := sdb.primaryHealth.GetStatus()
	if oldPrimaryStatus != newPrimaryStatus && sdb.config.OnHealthChange != nil {
		sdb.config.OnHealthChange("primary", oldPrimaryStatus, newPrimaryStatus)
	}

	// Check shadow
	if sdb.shadow != nil {
		shadowHealthy := sdb.checkDatabaseHealth(ctx, sdb.shadow, "shadow")
		oldShadowStatus := sdb.shadowHealth.GetStatus()

		if shadowHealthy {
			sdb.shadowHealth.updateStatus(StatusHealthy)
		} else {
			if sdb.shadowHealth.consecutiveFails >= sdb.config.MaxFailures {
				sdb.shadowHealth.updateStatus(StatusUnhealthy)
			} else {
				sdb.shadowHealth.updateStatus(StatusDegraded)
			}
		}

		newShadowStatus := sdb.shadowHealth.GetStatus()
		if oldShadowStatus != newShadowStatus && sdb.config.OnHealthChange != nil {
			sdb.config.OnHealthChange("shadow", oldShadowStatus, newShadowStatus)
		}
	}

	// Handle auto-failover and auto-failback
	if sdb.config.AutoFailover {
		sdb.handleAutoFailover()
	}
}

// checkDatabaseHealth performs health check on a database
func (sdb *ShadowDB) checkDatabaseHealth(ctx context.Context, db *sql.DB, name string) bool {
	if db == nil {
		return false
	}

	// Ping the database
	if err := db.PingContext(ctx); err != nil {
		return false
	}

	// Execute a simple query to verify connectivity
	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		return false
	}

	return result == 1
}

// handleAutoFailover handles automatic failover logic
func (sdb *ShadowDB) handleAutoFailover() {
	sdb.mu.RLock()
	currentlyPrimary := sdb.activePrimary
	primaryHealthy := sdb.primaryHealth.isHealthy()
	shadowHealthy := sdb.shadowHealth.isHealthy()
	sdb.mu.RUnlock()

	// Failover from primary to shadow
	if currentlyPrimary && !primaryHealthy && shadowHealthy {
		sdb.Failover()
	}

	// Failback from shadow to primary (auto-failback)
	if sdb.config.AutoFailback && !currentlyPrimary && primaryHealthy {
		sdb.Failback()
	}
}

// Failover manually triggers failover from primary to shadow
func (sdb *ShadowDB) Failover() error {
	sdb.failoverLock.Lock()
	defer sdb.failoverLock.Unlock()

	sdb.mu.Lock()
	defer sdb.mu.Unlock()

	// Check if already using shadow
	if !sdb.activePrimary {
		return nil // Already failed over
	}

	// Check if shadow is available
	if sdb.shadow == nil || !sdb.shadowHealth.isHealthy() {
		return ErrNoShadowDB
	}

	// Perform failover
	sdb.activePrimary = false

	if sdb.config.OnFailover != nil {
		go sdb.config.OnFailover("primary", "shadow")
	}

	return nil
}

// Failback manually triggers failback from shadow to primary
func (sdb *ShadowDB) Failback() error {
	sdb.failoverLock.Lock()
	defer sdb.failoverLock.Unlock()

	sdb.mu.Lock()
	defer sdb.mu.Unlock()

	// Check if already using primary
	if sdb.activePrimary {
		return nil // Already on primary
	}

	// Check if primary is available
	if sdb.primary == nil || !sdb.primaryHealth.isHealthy() {
		return ErrNoPrimaryDB
	}

	// Perform failback
	sdb.activePrimary = true

	if sdb.config.OnFailback != nil {
		go sdb.config.OnFailback()
	}

	return nil
}

// IsUsingPrimary returns true if currently using primary database
func (sdb *ShadowDB) IsUsingPrimary() bool {
	sdb.mu.RLock()
	defer sdb.mu.RUnlock()
	return sdb.activePrimary
}

// GetPrimaryHealth returns primary database health status
func (sdb *ShadowDB) GetPrimaryHealth() HealthStats {
	return sdb.primaryHealth.GetStats()
}

// GetShadowHealth returns shadow database health status
func (sdb *ShadowDB) GetShadowHealth() HealthStats {
	return sdb.shadowHealth.GetStats()
}

// GetStatus returns overall system status
func (sdb *ShadowDB) GetStatus() SystemStatus {
	sdb.mu.RLock()
	defer sdb.mu.RUnlock()

	return SystemStatus{
		ActiveDB:      sdb.getActiveDBName(),
		PrimaryHealth: sdb.primaryHealth.GetStats(),
		ShadowHealth:  sdb.shadowHealth.GetStats(),
		ReadStrategy:  sdb.config.ReadStrategy,
		WriteStrategy: sdb.config.WriteStrategy,
		AutoFailover:  sdb.config.AutoFailover,
		AutoFailback:  sdb.config.AutoFailback,
	}
}

// getActiveDBName returns name of active database
func (sdb *ShadowDB) getActiveDBName() string {
	if sdb.activePrimary {
		return "primary"
	}
	return "shadow"
}

// SystemStatus represents overall system status
type SystemStatus struct {
	ActiveDB      string
	PrimaryHealth HealthStats
	ShadowHealth  HealthStats
	ReadStrategy  ReadStrategy
	WriteStrategy WriteStrategy
	AutoFailover  bool
	AutoFailback  bool
}
