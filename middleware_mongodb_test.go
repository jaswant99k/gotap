package goTap

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// getMongoURI returns MongoDB URI from environment or empty string
func getMongoURI() string {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return "mongodb://localhost:27017"
	}
	return uri
}

// skipIfNoMongo skips the test if MongoDB is not available
func skipIfNoMongo(t *testing.T) *MongoClient {
	uri := getMongoURI()
	client, err := NewMongoClient(uri, "test_db")
	if err != nil {
		t.Skip("MongoDB not available, skipping test")
		return nil
	}
	return client
}

func TestNewMongoClient(t *testing.T) {
	client := skipIfNoMongo(t)
	if client == nil {
		return
	}
	defer client.Close()

	if client.Client == nil {
		t.Error("MongoDB client is nil")
	}

	if client.Database == nil {
		t.Error("MongoDB database is nil")
	}
}

func TestNewMongoClientFailure(t *testing.T) {
	// Try to connect to non-existent MongoDB server
	_, err := NewMongoClient("mongodb://localhost:99999", "test_db")
	if err == nil {
		t.Error("Expected error when connecting to non-existent MongoDB server")
	}
}

func TestMongoInjectAndGet(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	r := New()
	r.Use(MongoInject(mongoClient))

	r.GET("/test", func(c *Context) {
		client, ok := GetMongo(c)
		if !ok {
			t.Error("MongoDB client not found in context")
			c.JSON(500, H{"error": "MongoDB not available"})
			return
		}

		if client.Client == nil {
			t.Error("MongoDB client is nil")
		}

		c.JSON(200, H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMustGetMongo(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	r := New()
	r.Use(MongoInject(mongoClient))

	r.GET("/test", func(c *Context) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("MustGetMongo should not panic when MongoDB is available")
			}
		}()

		client := MustGetMongo(c)
		if client == nil {
			t.Error("Expected MongoDB client, got nil")
		}
		c.JSON(200, H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestMustGetMongoPanic(t *testing.T) {
	r := New()

	r.GET("/test", func(c *Context) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetMongo should panic when MongoDB is not available")
			}
		}()

		MustGetMongo(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestMongoHealthCheck(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	r := New()
	r.GET("/health", MongoHealthCheck(mongoClient))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !contains(w.Body.String(), "healthy") {
		t.Error("Expected healthy status in response")
	}
}

func TestMongoHealthCheckNoClient(t *testing.T) {
	r := New()
	r.GET("/health", MongoHealthCheck(nil))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 503 {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestMongoRepository(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	repo := NewMongoRepository(mongoClient, "products")
	ctx := context.Background()

	// Test InsertOne
	product := bson.M{
		"name":  "Test Product",
		"price": 29.99,
		"stock": 100,
	}

	result, err := repo.InsertOne(ctx, product)
	if err != nil {
		t.Errorf("Failed to insert document: %v", err)
	}

	if result.InsertedID == nil {
		t.Error("Expected inserted ID, got nil")
	}

	// Test FindOne
	var found bson.M
	findResult, _ := repo.FindOne(ctx, bson.M{"name": "Test Product"})
	err = findResult.Decode(&found)
	if err != nil {
		t.Errorf("Failed to find document: %v", err)
	}

	if found["name"] != "Test Product" {
		t.Errorf("Expected 'Test Product', got %v", found["name"])
	}

	// Test UpdateOne
	update := bson.M{"$set": bson.M{"price": 39.99}}
	updateResult, err := repo.UpdateOne(ctx, bson.M{"name": "Test Product"}, update)
	if err != nil {
		t.Errorf("Failed to update document: %v", err)
	}

	if updateResult.ModifiedCount != 1 {
		t.Errorf("Expected 1 modified document, got %d", updateResult.ModifiedCount)
	}

	// Test CountDocuments
	count, err := repo.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Errorf("Failed to count documents: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 document, got %d", count)
	}

	// Test DeleteOne
	deleteResult, err := repo.DeleteOne(ctx, bson.M{"name": "Test Product"})
	if err != nil {
		t.Errorf("Failed to delete document: %v", err)
	}

	if deleteResult.DeletedCount != 1 {
		t.Errorf("Expected 1 deleted document, got %d", deleteResult.DeletedCount)
	}
}

func TestMongoRepositoryInsertMany(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	repo := NewMongoRepository(mongoClient, "products")
	ctx := context.Background()

	products := []interface{}{
		bson.M{"name": "Product 1", "price": 10.99},
		bson.M{"name": "Product 2", "price": 20.99},
		bson.M{"name": "Product 3", "price": 30.99},
	}

	result, err := repo.InsertMany(ctx, products)
	if err != nil {
		t.Errorf("Failed to insert documents: %v", err)
	}

	if len(result.InsertedIDs) != 3 {
		t.Errorf("Expected 3 inserted documents, got %d", len(result.InsertedIDs))
	}
}

func TestMongoRepositoryFind(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	repo := NewMongoRepository(mongoClient, "products")
	ctx := context.Background()

	// Insert test data
	products := []interface{}{
		bson.M{"name": "Product 1", "price": 10.99},
		bson.M{"name": "Product 2", "price": 20.99},
		bson.M{"name": "Product 3", "price": 30.99},
	}
	repo.InsertMany(ctx, products)

	// Find all
	cursor, err := repo.Find(ctx, bson.M{})
	if err != nil {
		t.Errorf("Failed to find documents: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		t.Errorf("Failed to decode results: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(results))
	}
}

func TestMongoCache(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	cache := NewMongoCache(mongoClient, "cache", 1*time.Hour)
	ctx := context.Background()

	// Test Set
	err := cache.Set(ctx, "key1", "value1")
	if err != nil {
		t.Errorf("Failed to set cache: %v", err)
	}

	// Test Get
	value, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Failed to get cache: %v", err)
	}

	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// Test Delete
	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Errorf("Failed to delete cache: %v", err)
	}

	// Verify deletion
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}
}

func TestMongoCacheClear(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	cache := NewMongoCache(mongoClient, "cache", 1*time.Hour)
	ctx := context.Background()

	// Set multiple keys
	cache.Set(ctx, "key1", "value1")
	cache.Set(ctx, "key2", "value2")
	cache.Set(ctx, "key3", "value3")

	// Clear all
	err := cache.Clear(ctx)
	if err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}

	// Verify all keys are deleted
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("Expected error when getting cleared key")
	}
}

func TestMongoPagination(t *testing.T) {
	r := New()

	r.GET("/test", func(c *Context) {
		pagination := NewMongoPagination(c)

		if pagination.Page != 1 {
			t.Errorf("Expected page 1, got %d", pagination.Page)
		}

		if pagination.PageSize != 20 {
			t.Errorf("Expected page size 20, got %d", pagination.PageSize)
		}

		pagination.SetTotal(100)

		if pagination.Pages != 5 {
			t.Errorf("Expected 5 pages, got %d", pagination.Pages)
		}

		response := pagination.Response()
		if response["page"] != int64(1) {
			t.Error("Expected page 1 in response")
		}

		c.JSON(200, response)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMongoPaginationCustom(t *testing.T) {
	r := New()

	r.GET("/test", func(c *Context) {
		pagination := NewMongoPagination(c)

		// Test skip calculation for page 2 with page_size 10
		// page=2, page_size=10 should give skip = (2-1)*10 = 10
		skip := pagination.Skip()
		if skip != 10 {
			t.Errorf("Expected skip 10 for page 2, got %d", skip)
		}

		// Test with custom page size
		pagination.PageSize = 20
		skip = pagination.Skip()
		if skip != 20 {
			t.Errorf("Expected skip 20 for page 2 with page_size 20, got %d", skip)
		}

		c.JSON(200, H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test?page=2&page_size=10", nil)
	r.ServeHTTP(w, req)
}

func TestMongoAuditLog(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	auditLog := NewMongoAuditLog(mongoClient, "audit_log", false)

	r := New()
	r.Use(auditLog.Middleware())

	r.GET("/test", func(c *Context) {
		c.JSON(200, H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Give goroutine time to insert
	time.Sleep(100 * time.Millisecond)

	// Verify log entry was created
	collection := mongoClient.Collection("audit_log")
	count, err := collection.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		t.Errorf("Failed to count audit logs: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", count)
	}
}

func TestMongoRepositoryByID(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	repo := NewMongoRepository(mongoClient, "products")
	ctx := context.Background()

	// Insert with specific ID
	product := bson.M{
		"_id":   "prod123",
		"name":  "Test Product",
		"price": 29.99,
	}

	_, err := repo.InsertOne(ctx, product)
	if err != nil {
		t.Errorf("Failed to insert document: %v", err)
	}

	// Test FindByID
	var found bson.M
	err = repo.FindByID(ctx, "prod123").Decode(&found)
	if err != nil {
		t.Errorf("Failed to find by ID: %v", err)
	}

	if found["name"] != "Test Product" {
		t.Errorf("Expected 'Test Product', got %v", found["name"])
	}

	// Test UpdateByID
	update := bson.M{"$set": bson.M{"price": 39.99}}
	result, err := repo.UpdateByID(ctx, "prod123", update)
	if err != nil {
		t.Errorf("Failed to update by ID: %v", err)
	}

	if result.ModifiedCount != 1 {
		t.Errorf("Expected 1 modified, got %d", result.ModifiedCount)
	}

	// Test DeleteByID
	deleteResult, err := repo.DeleteByID(ctx, "prod123")
	if err != nil {
		t.Errorf("Failed to delete by ID: %v", err)
	}

	if deleteResult.DeletedCount != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleteResult.DeletedCount)
	}
}

func TestMongoRepositoryAggregate(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	repo := NewMongoRepository(mongoClient, "products")
	ctx := context.Background()

	// Insert test data
	products := []interface{}{
		bson.M{"category": "electronics", "price": 100.0},
		bson.M{"category": "electronics", "price": 200.0},
		bson.M{"category": "books", "price": 10.0},
	}
	repo.InsertMany(ctx, products)

	// Aggregate by category
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":   "$category",
			"total": bson.M{"$sum": "$price"},
		}}},
	}

	cursor, err := repo.Aggregate(ctx, pipeline)
	if err != nil {
		t.Errorf("Failed to aggregate: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		t.Errorf("Failed to decode results: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(results))
	}
}

func TestMongoTextSearch(t *testing.T) {
	mongoClient := skipIfNoMongo(t)
	if mongoClient == nil {
		return
	}
	defer mongoClient.Close()

	search := NewMongoTextSearch(mongoClient, "products")
	ctx := context.Background()

	// Create text index
	err := search.CreateTextIndex(ctx, "name", "description")
	if err != nil {
		t.Errorf("Failed to create text index: %v", err)
	}

	// Insert test data
	repo := search.repository
	products := []interface{}{
		bson.M{"name": "Laptop Computer", "description": "High performance laptop"},
		bson.M{"name": "Desktop Computer", "description": "Gaming desktop"},
		bson.M{"name": "Tablet Device", "description": "Portable tablet"},
	}
	repo.InsertMany(ctx, products)

	// Search for "computer"
	cursor, err := search.Search(ctx, "computer")
	if err != nil {
		t.Errorf("Failed to search: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		t.Errorf("Failed to decode results: %v", err)
	}

	// Should find 2 products with "computer"
	if len(results) < 2 {
		t.Errorf("Expected at least 2 search results, got %d", len(results))
	}
}
