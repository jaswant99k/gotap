package goTap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestNewInMemoryVectorStore(t *testing.T) {
	store := NewInMemoryVectorStore()
	if store == nil {
		t.Fatal("Expected vector store, got nil")
	}

	if store.vectors == nil {
		t.Error("Expected vectors map to be initialized")
	}
}

func TestVectorInsert(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	documents := []*VectorDocument{
		{
			ID:     "vec1",
			Vector: Vector{1.0, 2.0, 3.0},
			Metadata: map[string]interface{}{
				"name": "Item 1",
			},
		},
		{
			ID:     "vec2",
			Vector: Vector{4.0, 5.0, 6.0},
			Metadata: map[string]interface{}{
				"name": "Item 2",
			},
		},
	}

	err := store.Insert(ctx, documents)
	if err != nil {
		t.Errorf("Failed to insert vectors: %v", err)
	}

	if len(store.vectors) != 2 {
		t.Errorf("Expected 2 vectors, got %d", len(store.vectors))
	}
}

func TestVectorGet(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	doc := &VectorDocument{
		ID:     "vec1",
		Vector: Vector{1.0, 2.0, 3.0},
		Metadata: map[string]interface{}{
			"name": "Test Item",
		},
	}

	store.Insert(ctx, []*VectorDocument{doc})

	// Test Get existing
	retrieved, err := store.Get(ctx, "vec1")
	if err != nil {
		t.Errorf("Failed to get vector: %v", err)
	}

	if retrieved.ID != "vec1" {
		t.Errorf("Expected ID vec1, got %s", retrieved.ID)
	}

	// Test Get non-existing
	_, err = store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent vector")
	}
}

func TestVectorUpdate(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	doc := &VectorDocument{
		ID:     "vec1",
		Vector: Vector{1.0, 2.0, 3.0},
		Metadata: map[string]interface{}{
			"name": "Original",
		},
	}

	store.Insert(ctx, []*VectorDocument{doc})

	// Update
	updated := &VectorDocument{
		ID:     "vec1",
		Vector: Vector{4.0, 5.0, 6.0},
		Metadata: map[string]interface{}{
			"name": "Updated",
		},
	}

	err := store.Update(ctx, updated)
	if err != nil {
		t.Errorf("Failed to update vector: %v", err)
	}

	// Verify update
	retrieved, _ := store.Get(ctx, "vec1")
	if retrieved.Metadata["name"] != "Updated" {
		t.Error("Vector was not updated")
	}

	// Test update non-existing
	nonExistent := &VectorDocument{ID: "nonexistent", Vector: Vector{1.0}}
	err = store.Update(ctx, nonExistent)
	if err == nil {
		t.Error("Expected error when updating non-existent vector")
	}
}

func TestVectorDelete(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	documents := []*VectorDocument{
		{ID: "vec1", Vector: Vector{1.0, 2.0, 3.0}},
		{ID: "vec2", Vector: Vector{4.0, 5.0, 6.0}},
		{ID: "vec3", Vector: Vector{7.0, 8.0, 9.0}},
	}

	store.Insert(ctx, documents)

	// Delete two vectors
	err := store.Delete(ctx, []string{"vec1", "vec3"})
	if err != nil {
		t.Errorf("Failed to delete vectors: %v", err)
	}

	if len(store.vectors) != 1 {
		t.Errorf("Expected 1 vector remaining, got %d", len(store.vectors))
	}

	// Verify vec2 still exists
	_, err = store.Get(ctx, "vec2")
	if err != nil {
		t.Error("vec2 should still exist")
	}

	// Verify vec1 is deleted
	_, err = store.Get(ctx, "vec1")
	if err == nil {
		t.Error("vec1 should be deleted")
	}
}

func TestVectorSearch(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	// Insert test vectors
	documents := []*VectorDocument{
		{ID: "vec1", Vector: Vector{1.0, 0.0, 0.0}},
		{ID: "vec2", Vector: Vector{0.9, 0.1, 0.0}}, // Similar to vec1
		{ID: "vec3", Vector: Vector{0.0, 1.0, 0.0}}, // Orthogonal to vec1
	}

	store.Insert(ctx, documents)

	// Search for vectors similar to [1.0, 0.0, 0.0]
	queryVector := Vector{1.0, 0.0, 0.0}
	results, err := store.Search(ctx, queryVector, 2)
	if err != nil {
		t.Errorf("Failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// First result should be vec1 (perfect match)
	if results[0].Document.ID != "vec1" {
		t.Errorf("Expected vec1 as first result, got %s", results[0].Document.ID)
	}

	// Second result should be vec2 (similar)
	if results[1].Document.ID != "vec2" {
		t.Errorf("Expected vec2 as second result, got %s", results[1].Document.ID)
	}
}

func TestCosineSimilarity(t *testing.T) {
	// Test identical vectors
	v1 := Vector{1.0, 2.0, 3.0}
	v2 := Vector{1.0, 2.0, 3.0}
	similarity := CosineSimilarity(v1, v2)
	if similarity < 0.99 {
		t.Errorf("Expected similarity ~1.0 for identical vectors, got %f", similarity)
	}

	// Test orthogonal vectors
	v3 := Vector{1.0, 0.0, 0.0}
	v4 := Vector{0.0, 1.0, 0.0}
	similarity = CosineSimilarity(v3, v4)
	if similarity > 0.01 {
		t.Errorf("Expected similarity ~0.0 for orthogonal vectors, got %f", similarity)
	}

	// Test different length vectors
	v5 := Vector{1.0, 2.0}
	v6 := Vector{1.0, 2.0, 3.0}
	similarity = CosineSimilarity(v5, v6)
	if similarity != 0 {
		t.Errorf("Expected 0 for different length vectors, got %f", similarity)
	}
}

func TestEuclideanDistance(t *testing.T) {
	// Test identical vectors
	v1 := Vector{1.0, 2.0, 3.0}
	v2 := Vector{1.0, 2.0, 3.0}
	distance := EuclideanDistance(v1, v2)
	if distance > 0.01 {
		t.Errorf("Expected distance ~0.0 for identical vectors, got %f", distance)
	}

	// Test known distance
	v3 := Vector{0.0, 0.0, 0.0}
	v4 := Vector{3.0, 4.0, 0.0}
	distance = EuclideanDistance(v3, v4)
	// 3-4-5 triangle, distance should be 5.0
	if distance < 4.99 || distance > 5.01 {
		t.Errorf("Expected distance ~5.0, got %f", distance)
	}
}

func TestDotProduct(t *testing.T) {
	v1 := Vector{1.0, 2.0, 3.0}
	v2 := Vector{4.0, 5.0, 6.0}
	// 1*4 + 2*5 + 3*6 = 4 + 10 + 18 = 32
	product := DotProduct(v1, v2)
	if product != 32.0 {
		t.Errorf("Expected dot product 32.0, got %f", product)
	}

	// Test with different lengths
	v3 := Vector{1.0, 2.0}
	v4 := Vector{1.0, 2.0, 3.0}
	product = DotProduct(v3, v4)
	if product != 0 {
		t.Errorf("Expected 0 for different length vectors, got %f", product)
	}
}

func TestNormalize(t *testing.T) {
	v := Vector{3.0, 4.0, 0.0}
	normalized := Normalize(v)

	// Check length is 1
	var length float32
	for _, val := range normalized {
		length += val * val
	}
	length = float32(0.5) * length * 2.0 // Avoiding math import

	if normalized[0] < 0.59 || normalized[0] > 0.61 {
		t.Errorf("Expected normalized[0] ~0.6, got %f", normalized[0])
	}

	if normalized[1] < 0.79 || normalized[1] > 0.81 {
		t.Errorf("Expected normalized[1] ~0.8, got %f", normalized[1])
	}
}

func TestVectorInjectAndGet(t *testing.T) {
	store := NewInMemoryVectorStore()

	r := New()
	r.Use(VectorInject(store))

	r.GET("/test", func(c *Context) {
		vs, ok := GetVectorStore(c)
		if !ok {
			t.Error("VectorStore not found in context")
			c.JSON(500, H{"error": "VectorStore not available"})
			return
		}

		if vs == nil {
			t.Error("VectorStore is nil")
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

func TestMustGetVectorStore(t *testing.T) {
	store := NewInMemoryVectorStore()

	r := New()
	r.Use(VectorInject(store))

	r.GET("/test", func(c *Context) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("MustGetVectorStore should not panic when store is available")
			}
		}()

		vs := MustGetVectorStore(c)
		if vs == nil {
			t.Error("Expected VectorStore, got nil")
		}
		c.JSON(200, H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestMustGetVectorStorePanic(t *testing.T) {
	r := New()

	r.GET("/test", func(c *Context) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetVectorStore should panic when store is not available")
			}
		}()

		MustGetVectorStore(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestVectorSearchHandler(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	// Insert test data
	documents := []*VectorDocument{
		{ID: "prod1", Vector: Vector{1.0, 0.0, 0.0}},
		{ID: "prod2", Vector: Vector{0.9, 0.1, 0.0}},
		{ID: "prod3", Vector: Vector{0.0, 1.0, 0.0}},
	}
	store.Insert(ctx, documents)

	r := New()
	r.Use(VectorInject(store))
	r.POST("/search", VectorSearchHandler())

	// Test search
	searchReq := VectorSearchRequest{
		Vector: Vector{1.0, 0.0, 0.0},
		Limit:  2,
	}

	body, _ := json.Marshal(searchReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	results := response["results"].([]interface{})
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestVectorInsertHandler(t *testing.T) {
	store := NewInMemoryVectorStore()

	r := New()
	r.Use(VectorInject(store))
	r.POST("/vectors", VectorInsertHandler())

	documents := []*VectorDocument{
		{ID: "vec1", Vector: Vector{1.0, 2.0, 3.0}},
		{ID: "vec2", Vector: Vector{4.0, 5.0, 6.0}},
	}

	body, _ := json.Marshal(documents)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/vectors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify insertion
	if len(store.vectors) != 2 {
		t.Errorf("Expected 2 vectors in store, got %d", len(store.vectors))
	}
}

func TestVectorDeleteHandler(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	// Insert test data
	documents := []*VectorDocument{
		{ID: "vec1", Vector: Vector{1.0, 2.0, 3.0}},
		{ID: "vec2", Vector: Vector{4.0, 5.0, 6.0}},
	}
	store.Insert(ctx, documents)

	r := New()
	r.Use(VectorInject(store))
	r.DELETE("/vectors", VectorDeleteHandler())

	deleteReq := map[string]interface{}{
		"ids": []string{"vec1"},
	}

	body, _ := json.Marshal(deleteReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/vectors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify deletion
	if len(store.vectors) != 1 {
		t.Errorf("Expected 1 vector remaining, got %d", len(store.vectors))
	}
}

func TestVectorGetHandler(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	doc := &VectorDocument{
		ID:     "vec1",
		Vector: Vector{1.0, 2.0, 3.0},
		Metadata: map[string]interface{}{
			"name": "Test Vector",
		},
	}
	store.Insert(ctx, []*VectorDocument{doc})

	r := New()
	r.Use(VectorInject(store))
	r.GET("/vectors/:id", VectorGetHandler())

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/vectors/vec1", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response VectorDocument
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.ID != "vec1" {
		t.Errorf("Expected ID vec1, got %s", response.ID)
	}
}

func TestProductRecommender(t *testing.T) {
	store := NewInMemoryVectorStore()
	recommender := NewProductRecommender(store)
	ctx := context.Background()

	// Add products
	products := []*ProductEmbedding{
		{
			ProductID: "prod1",
			Name:      "Laptop",
			Category:  "electronics",
			Price:     999.99,
			Vector:    Vector{1.0, 0.0, 0.0},
		},
		{
			ProductID: "prod2",
			Name:      "Desktop",
			Category:  "electronics",
			Price:     1299.99,
			Vector:    Vector{0.9, 0.1, 0.0},
		},
		{
			ProductID: "prod3",
			Name:      "T-Shirt",
			Category:  "clothing",
			Price:     19.99,
			Vector:    Vector{0.0, 1.0, 0.0},
		},
	}

	for _, prod := range products {
		err := recommender.AddProduct(ctx, prod)
		if err != nil {
			t.Errorf("Failed to add product: %v", err)
		}
	}

	// Get similar products to prod1 (Laptop)
	similar, err := recommender.GetSimilarProducts(ctx, "prod1", 2)
	if err != nil {
		t.Errorf("Failed to get similar products: %v", err)
	}

	if len(similar) == 0 {
		t.Error("Expected similar products, got none")
	}

	// First similar should be prod2 (Desktop)
	if len(similar) > 0 && similar[0].ProductID != "prod2" {
		t.Errorf("Expected prod2 as first similar, got %s", similar[0].ProductID)
	}
}

func TestVectorJSONConversion(t *testing.T) {
	vector := Vector{1.0, 2.0, 3.0}

	// To JSON
	jsonStr := VectorToJSON(vector)
	if jsonStr == "" {
		t.Error("Expected JSON string, got empty")
	}

	// From JSON
	converted, err := JSONToVector(jsonStr)
	if err != nil {
		t.Errorf("Failed to convert from JSON: %v", err)
	}

	if len(converted) != len(vector) {
		t.Errorf("Expected length %d, got %d", len(vector), len(converted))
	}

	for i := range vector {
		if converted[i] != vector[i] {
			t.Errorf("Expected %f at index %d, got %f", vector[i], i, converted[i])
		}
	}
}
