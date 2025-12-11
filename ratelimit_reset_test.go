package goTap

import (
	"testing"
	"time"
)

// Test rate limiter Reset method directly
func TestRateLimiterResetDirect(t *testing.T) {
	store := newInMemoryStore()
	
	// Add an entry
	count, _, err := store.Increment("test-key", time.Minute)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
	
	// Increment again
	count, _, err = store.Increment("test-key", time.Minute)
	if err != nil {
		t.Fatalf("Failed to increment again: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
	
	// Reset the key
	err = store.Reset("test-key")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	
	// After reset, count should start at 1 again
	count, _, err = store.Increment("test-key", time.Minute)
	if err != nil {
		t.Fatalf("Failed to increment after reset: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 after reset, got %d", count)
	}
}

// Test Reset on non-existent key (should not error)
func TestRateLimiterResetNonExistent(t *testing.T) {
	store := newInMemoryStore()
	
	// Reset a key that doesn't exist
	err := store.Reset("nonexistent-key")
	if err != nil {
		t.Errorf("Reset should not error on non-existent key, got: %v", err)
	}
}

// Test multiple Reset calls
func TestRateLimiterMultipleReset(t *testing.T) {
	store := newInMemoryStore()
	
	// Add entries
	store.Increment("key1", time.Minute)
	store.Increment("key2", time.Minute)
	store.Increment("key3", time.Minute)
	
	// Reset all keys
	if err := store.Reset("key1"); err != nil {
		t.Errorf("Reset key1 failed: %v", err)
	}
	if err := store.Reset("key2"); err != nil {
		t.Errorf("Reset key2 failed: %v", err)
	}
	if err := store.Reset("key3"); err != nil {
		t.Errorf("Reset key3 failed: %v", err)
	}
	
	// All keys should start fresh
	count1, _, _ := store.Increment("key1", time.Minute)
	count2, _, _ := store.Increment("key2", time.Minute)
	count3, _, _ := store.Increment("key3", time.Minute)
	
	if count1 != 1 || count2 != 1 || count3 != 1 {
		t.Errorf("Expected all counts to be 1 after reset, got %d, %d, %d", count1, count2, count3)
	}
}
