// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var transactionCounter uint64

// TransactionIDConfig holds TransactionID middleware configuration
type TransactionIDConfig struct {
	// Generator defines a function to generate transaction IDs
	// Default: uses timestamp + random bytes + counter
	Generator func() string

	// HeaderName is the name of the header to set
	// Default: X-Transaction-ID
	HeaderName string

	// ContextKey is the key to store transaction ID in context
	// Default: transaction_id
	ContextKey string

	// IncludeInResponse determines if the transaction ID should be added to response headers
	// Default: true
	IncludeInResponse bool
}

// TransactionID returns a middleware that generates unique transaction IDs for each request
func TransactionID() HandlerFunc {
	return TransactionIDWithConfig(TransactionIDConfig{})
}

// TransactionIDWithConfig returns a TransactionID middleware with config
func TransactionIDWithConfig(config TransactionIDConfig) HandlerFunc {
	if config.Generator == nil {
		config.Generator = defaultTransactionIDGenerator
	}

	if config.HeaderName == "" {
		config.HeaderName = "X-Transaction-ID"
	}

	if config.ContextKey == "" {
		config.ContextKey = "transaction_id"
	}

	config.IncludeInResponse = true

	return func(c *Context) {
		// Check if transaction ID already exists in request header
		txID := c.Request.Header.Get(config.HeaderName)

		// If not, generate new one
		if txID == "" {
			txID = config.Generator()
		}

		// Store in context
		c.Set(config.ContextKey, txID)

		// Add to response header
		if config.IncludeInResponse {
			c.Writer.Header().Set(config.HeaderName, txID)
		}

		c.Next()
	}
}

// defaultTransactionIDGenerator generates a unique transaction ID
// Format: YYYYMMDD-HHMMSS-COUNTER-RANDOM
func defaultTransactionIDGenerator() string {
	// Timestamp
	now := time.Now()
	timestamp := now.Format("20060102-150405")

	// Counter
	counter := atomic.AddUint64(&transactionCounter, 1)

	// Random bytes
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("%s-%d-%s", timestamp, counter, randomHex)
}

// GetTransactionID retrieves transaction ID from context
func GetTransactionID(c *Context) string {
	txID, exists := c.Get("transaction_id")
	if !exists {
		return ""
	}
	if id, ok := txID.(string); ok {
		return id
	}
	return ""
}

// UUIDTransactionIDGenerator generates UUID-like transaction IDs
func UUIDTransactionIDGenerator() string {
	b := make([]byte, 16)
	rand.Read(b)

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ShortTransactionIDGenerator generates short transaction IDs (12 chars)
func ShortTransactionIDGenerator() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// POSTransactionIDGenerator generates POS-specific transaction IDs
// Format: POS-TERMINALID-TIMESTAMP-COUNTER
func POSTransactionIDGenerator(terminalID string) func() string {
	return func() string {
		now := time.Now()
		timestamp := now.Format("20060102150405")
		counter := atomic.AddUint64(&transactionCounter, 1)

		return fmt.Sprintf("POS-%s-%s-%06d", terminalID, timestamp, counter%1000000)
	}
}
