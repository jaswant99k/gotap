// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"database/sql"

	"github.com/jaswant99k/gotap/shadowdb"
)

// ShadowDBMiddleware returns a middleware that injects Shadow DB into context
func ShadowDBMiddleware(sdb *shadowdb.ShadowDB) HandlerFunc {
	return func(c *Context) {
		c.Set(shadowdb.ContextKeyShadowDB, sdb)

		// Pre-fetch read and write connections for this request
		readDB, _ := sdb.Read()
		writeDB, _ := sdb.Write()

		c.Set(shadowdb.ContextKeyReadDB, readDB)
		c.Set(shadowdb.ContextKeyWriteDB, writeDB)

		c.Next()
	}
}

// GetShadowDB retrieves Shadow DB from context
func GetShadowDB(c *Context) (*shadowdb.ShadowDB, bool) {
	sdb, exists := c.Get(shadowdb.ContextKeyShadowDB)
	if !exists {
		return nil, false
	}
	shadowDB, ok := sdb.(*shadowdb.ShadowDB)
	return shadowDB, ok
}

// GetReadDB retrieves read database connection from context
func GetReadDB(c *Context) (*sql.DB, bool) {
	db, exists := c.Get(shadowdb.ContextKeyReadDB)
	if !exists {
		return nil, false
	}
	readDB, ok := db.(*sql.DB)
	return readDB, ok
}

// GetWriteDB retrieves write database connection from context
func GetWriteDB(c *Context) (*sql.DB, bool) {
	db, exists := c.Get(shadowdb.ContextKeyWriteDB)
	if !exists {
		return nil, false
	}
	writeDB, ok := db.(*sql.DB)
	return writeDB, ok
}

// DBHealthCheck returns a middleware that checks database health
func DBHealthCheck() HandlerFunc {
	return func(c *Context) {
		sdb, exists := GetShadowDB(c)
		if !exists {
			c.Next()
			return
		}

		status := sdb.GetStatus()

		// Add health information to response headers
		c.Header("X-DB-Active", status.ActiveDB)
		c.Header("X-DB-Primary-Status", string(status.PrimaryHealth.Status))

		if status.ShadowHealth.Status != "" {
			c.Header("X-DB-Shadow-Status", string(status.ShadowHealth.Status))
		}

		c.Next()
	}
}

// RequireHealthyDB returns a middleware that requires a healthy database
func RequireHealthyDB() HandlerFunc {
	return func(c *Context) {
		sdb, exists := GetShadowDB(c)
		if !exists {
			c.JSON(500, H{
				"error":   "Internal Server Error",
				"message": "Database not configured",
			})
			c.Abort()
			return
		}

		// Check if we can get a write connection
		_, err := sdb.Write()
		if err != nil {
			c.JSON(503, H{
				"error":   "Service Unavailable",
				"message": "Database is currently unavailable",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
