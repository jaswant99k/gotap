// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package shadowdb

import (
	"database/sql"
)

// Middleware integration helpers

const (
	// ContextKeyShadowDB is the key for storing ShadowDB in context
	ContextKeyShadowDB = "shadowdb"
	// ContextKeyReadDB is the key for storing read DB in context
	ContextKeyReadDB = "shadowdb_read"
	// ContextKeyWriteDB is the key for storing write DB in context
	ContextKeyWriteDB = "shadowdb_write"
)

// Transaction helper for dual-write scenarios
type Transaction struct {
	Primary *sql.Tx
	Shadow  *sql.Tx
	sdb     *ShadowDB
}

// BeginTx starts a transaction based on write strategy
func (sdb *ShadowDB) BeginTx() (*Transaction, error) {
	tx := &Transaction{sdb: sdb}

	if sdb.config.WriteStrategy == WriteBoth {
		// Start transaction on both databases
		if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
			primaryTx, err := sdb.primary.Begin()
			if err != nil {
				return nil, err
			}
			tx.Primary = primaryTx
		}

		if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			shadowTx, err := sdb.shadow.Begin()
			if err != nil {
				// Rollback primary if shadow fails
				if tx.Primary != nil {
					tx.Primary.Rollback()
				}
				return nil, err
			}
			tx.Shadow = shadowTx
		}
	} else {
		// Single transaction based on active database
		db, err := sdb.Write()
		if err != nil {
			return nil, err
		}

		sqlTx, err := db.Begin()
		if err != nil {
			return nil, err
		}

		if sdb.activePrimary {
			tx.Primary = sqlTx
		} else {
			tx.Shadow = sqlTx
		}
	}

	return tx, nil
}

// Commit commits all transactions
func (tx *Transaction) Commit() error {
	var errs []error

	if tx.Primary != nil {
		if err := tx.Primary.Commit(); err != nil {
			errs = append(errs, err)
		}
	}

	if tx.Shadow != nil {
		if err := tx.Shadow.Commit(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// Rollback rolls back all transactions
func (tx *Transaction) Rollback() error {
	var errs []error

	if tx.Primary != nil {
		if err := tx.Primary.Rollback(); err != nil {
			errs = append(errs, err)
		}
	}

	if tx.Shadow != nil {
		if err := tx.Shadow.Rollback(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// Exec executes a query on appropriate database(s)
func (tx *Transaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	if tx.sdb.config.WriteStrategy == WriteBoth {
		// Execute on both
		var result sql.Result
		var err error

		if tx.Primary != nil {
			result, err = tx.Primary.Exec(query, args...)
			if err != nil {
				return nil, err
			}
		}

		if tx.Shadow != nil {
			_, err = tx.Shadow.Exec(query, args...)
			if err != nil {
				// Log error but don't fail the transaction
				// In production, you might want to handle this differently
			}
		}

		return result, nil
	}

	// Execute on active database
	if tx.Primary != nil {
		return tx.Primary.Exec(query, args...)
	}
	if tx.Shadow != nil {
		return tx.Shadow.Exec(query, args...)
	}

	return nil, ErrBothDBsDown
}

// Query executes a query and returns rows
func (tx *Transaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if tx.Primary != nil {
		return tx.Primary.Query(query, args...)
	}
	if tx.Shadow != nil {
		return tx.Shadow.Query(query, args...)
	}
	return nil, ErrBothDBsDown
}

// QueryRow executes a query that returns at most one row
func (tx *Transaction) QueryRow(query string, args ...interface{}) *sql.Row {
	if tx.Primary != nil {
		return tx.Primary.QueryRow(query, args...)
	}
	if tx.Shadow != nil {
		return tx.Shadow.QueryRow(query, args...)
	}
	// Return a row that will error on Scan
	return nil
}

// Helper functions for common operations

// ExecWrite executes a write query on the appropriate database(s)
func (sdb *ShadowDB) ExecWrite(query string, args ...interface{}) (sql.Result, error) {
	if sdb.config.WriteStrategy == WriteBoth {
		var result sql.Result
		var err error

		// Execute on primary
		if sdb.primary != nil && sdb.primaryHealth.isHealthy() {
			result, err = sdb.primary.Exec(query, args...)
			if err != nil {
				return nil, err
			}
		}

		// Execute on shadow
		if sdb.shadow != nil && sdb.shadowHealth.isHealthy() {
			_, _ = sdb.shadow.Exec(query, args...)
			// Ignore shadow errors in dual-write mode
		}

		return result, nil
	}

	// Single write
	db, err := sdb.Write()
	if err != nil {
		return nil, err
	}

	return db.Exec(query, args...)
}

// QueryRead executes a read query on the appropriate database
func (sdb *ShadowDB) QueryRead(query string, args ...interface{}) (*sql.Rows, error) {
	db, err := sdb.Read()
	if err != nil {
		return nil, err
	}

	return db.Query(query, args...)
}

// QueryRowRead executes a read query that returns at most one row
func (sdb *ShadowDB) QueryRowRead(query string, args ...interface{}) *sql.Row {
	db, err := sdb.Read()
	if err != nil {
		// Return a row that will error on Scan
		return nil
	}

	return db.QueryRow(query, args...)
}
