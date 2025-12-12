package goTap

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// Vector represents an embedding vector
type Vector []float32

// VectorDocument represents a document with vector embedding
type VectorDocument struct {
	ID       string                 `json:"id"`
	Vector   Vector                 `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
}

// VectorSearchResult represents a search result with similarity score
type VectorSearchResult struct {
	Document *VectorDocument `json:"document"`
	Score    float32         `json:"score"`
	Distance float32         `json:"distance"`
}

// VectorStore interface for vector database operations
type VectorStore interface {
	// Insert adds vectors to the store
	Insert(ctx context.Context, documents []*VectorDocument) error

	// Search performs similarity search
	Search(ctx context.Context, queryVector Vector, limit int) ([]*VectorSearchResult, error)

	// Delete removes vectors by ID
	Delete(ctx context.Context, ids []string) error

	// Get retrieves a vector by ID
	Get(ctx context.Context, id string) (*VectorDocument, error)

	// Update updates a vector
	Update(ctx context.Context, document *VectorDocument) error
}

// InMemoryVectorStore implements VectorStore in memory (for testing/demo)
type InMemoryVectorStore struct {
	vectors map[string]*VectorDocument
}

// NewInMemoryVectorStore creates a new in-memory vector store
func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		vectors: make(map[string]*VectorDocument),
	}
}

// Insert adds vectors to the store
func (s *InMemoryVectorStore) Insert(ctx context.Context, documents []*VectorDocument) error {
	for _, doc := range documents {
		s.vectors[doc.ID] = doc
	}
	return nil
}

// Search performs similarity search using cosine similarity
func (s *InMemoryVectorStore) Search(ctx context.Context, queryVector Vector, limit int) ([]*VectorSearchResult, error) {
	if len(s.vectors) == 0 {
		return []*VectorSearchResult{}, nil
	}

	results := make([]*VectorSearchResult, 0, len(s.vectors))

	for _, doc := range s.vectors {
		similarity := CosineSimilarity(queryVector, doc.Vector)
		distance := 1.0 - similarity

		results = append(results, &VectorSearchResult{
			Document: doc,
			Score:    similarity,
			Distance: distance,
		})
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return top results
	if limit > len(results) {
		limit = len(results)
	}

	return results[:limit], nil
}

// Delete removes vectors by ID
func (s *InMemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		delete(s.vectors, id)
	}
	return nil
}

// Get retrieves a vector by ID
func (s *InMemoryVectorStore) Get(ctx context.Context, id string) (*VectorDocument, error) {
	doc, exists := s.vectors[id]
	if !exists {
		return nil, fmt.Errorf("vector not found: %s", id)
	}
	return doc, nil
}

// Update updates a vector
func (s *InMemoryVectorStore) Update(ctx context.Context, document *VectorDocument) error {
	if _, exists := s.vectors[document.ID]; !exists {
		return fmt.Errorf("vector not found: %s", document.ID)
	}
	s.vectors[document.ID] = document
	return nil
}

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b Vector) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EuclideanDistance calculates Euclidean distance between two vectors
func EuclideanDistance(a, b Vector) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// DotProduct calculates dot product of two vectors
func DotProduct(a, b Vector) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// VectorInject injects vector store into context for use in handlers
func VectorInject(store VectorStore) HandlerFunc {
	return func(c *Context) {
		c.Set("vectorstore", store)
		c.Next()
	}
}

// GetVectorStore retrieves vector store from context
func GetVectorStore(c *Context) (VectorStore, bool) {
	store, exists := c.Get("vectorstore")
	if !exists {
		return nil, false
	}
	vectorStore, ok := store.(VectorStore)
	return vectorStore, ok
}

// MustGetVectorStore retrieves vector store from context or panics
func MustGetVectorStore(c *Context) VectorStore {
	store, ok := GetVectorStore(c)
	if !ok {
		panic("VectorStore not found in context")
	}
	return store
}

// VectorSearchRequest represents a search request
type VectorSearchRequest struct {
	Vector   Vector                 `json:"vector" binding:"required"`
	Limit    int                    `json:"limit"`
	Filter   map[string]interface{} `json:"filter"`
	MinScore float32                `json:"min_score"`
}

// VectorSearchHandler creates a handler for vector search
func VectorSearchHandler() HandlerFunc {
	return func(c *Context) {
		store := MustGetVectorStore(c)

		var req VectorSearchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, H{"error": err.Error()})
			return
		}

		// Default limit
		if req.Limit == 0 {
			req.Limit = 10
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := store.Search(ctx, req.Vector, req.Limit)
		if err != nil {
			c.JSON(500, H{"error": "Search failed", "details": err.Error()})
			return
		}

		// Apply min score filter
		if req.MinScore > 0 {
			filtered := make([]*VectorSearchResult, 0)
			for _, result := range results {
				if result.Score >= req.MinScore {
					filtered = append(filtered, result)
				}
			}
			results = filtered
		}

		c.JSON(200, H{
			"results": results,
			"count":   len(results),
		})
	}
}

// VectorInsertHandler creates a handler for inserting vectors
func VectorInsertHandler() HandlerFunc {
	return func(c *Context) {
		store := MustGetVectorStore(c)

		var documents []*VectorDocument
		if err := c.ShouldBindJSON(&documents); err != nil {
			c.JSON(400, H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := store.Insert(ctx, documents); err != nil {
			c.JSON(500, H{"error": "Insert failed", "details": err.Error()})
			return
		}

		c.JSON(201, H{
			"message": "Vectors inserted successfully",
			"count":   len(documents),
		})
	}
}

// VectorDeleteHandler creates a handler for deleting vectors
func VectorDeleteHandler() HandlerFunc {
	return func(c *Context) {
		store := MustGetVectorStore(c)

		var req struct {
			IDs []string `json:"ids" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := store.Delete(ctx, req.IDs); err != nil {
			c.JSON(500, H{"error": "Delete failed", "details": err.Error()})
			return
		}

		c.JSON(200, H{
			"message": "Vectors deleted successfully",
			"count":   len(req.IDs),
		})
	}
}

// VectorGetHandler creates a handler for getting a vector by ID
func VectorGetHandler() HandlerFunc {
	return func(c *Context) {
		store := MustGetVectorStore(c)
		id := c.Param("id")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		document, err := store.Get(ctx, id)
		if err != nil {
			c.JSON(404, H{"error": "Vector not found", "details": err.Error()})
			return
		}

		c.JSON(200, document)
	}
}

// ProductEmbedding represents a product with its vector embedding
type ProductEmbedding struct {
	ProductID   string  `json:"product_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Vector      Vector  `json:"vector"`
}

// ProductRecommender provides product recommendations using vector similarity
type ProductRecommender struct {
	store VectorStore
}

// NewProductRecommender creates a new product recommender
func NewProductRecommender(store VectorStore) *ProductRecommender {
	return &ProductRecommender{
		store: store,
	}
}

// AddProduct adds a product to the recommender
func (pr *ProductRecommender) AddProduct(ctx context.Context, product *ProductEmbedding) error {
	doc := &VectorDocument{
		ID:     product.ProductID,
		Vector: product.Vector,
		Metadata: map[string]interface{}{
			"name":        product.Name,
			"description": product.Description,
			"category":    product.Category,
			"price":       product.Price,
		},
	}

	return pr.store.Insert(ctx, []*VectorDocument{doc})
}

// GetSimilarProducts finds products similar to the given product
func (pr *ProductRecommender) GetSimilarProducts(ctx context.Context, productID string, limit int) ([]*ProductEmbedding, error) {
	// Get the product vector
	doc, err := pr.store.Get(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Search for similar products
	results, err := pr.store.Search(ctx, doc.Vector, limit+1) // +1 to exclude self
	if err != nil {
		return nil, err
	}

	// Convert results to product embeddings
	products := make([]*ProductEmbedding, 0, len(results))
	for _, result := range results {
		// Skip the product itself
		if result.Document.ID == productID {
			continue
		}

		product := &ProductEmbedding{
			ProductID:   result.Document.ID,
			Name:        getStringMetadata(result.Document.Metadata, "name"),
			Description: getStringMetadata(result.Document.Metadata, "description"),
			Category:    getStringMetadata(result.Document.Metadata, "category"),
			Price:       getFloatMetadata(result.Document.Metadata, "price"),
			Vector:      result.Document.Vector,
		}

		products = append(products, product)
	}

	return products, nil
}

// Helper functions
func getStringMetadata(metadata map[string]interface{}, key string) string {
	if val, ok := metadata[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloatMetadata(metadata map[string]interface{}, key string) float64 {
	if val, ok := metadata[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

// VectorMiddleware provides common vector operations middleware
type VectorMiddleware struct {
	store VectorStore
}

// NewVectorMiddleware creates a new vector middleware
func NewVectorMiddleware(store VectorStore) *VectorMiddleware {
	return &VectorMiddleware{store: store}
}

// SmartSearch middleware adds AI-powered search to endpoints
func (vm *VectorMiddleware) SmartSearch() HandlerFunc {
	return func(c *Context) {
		// Get search query
		query := c.Query("q")
		if query == "" {
			c.Next()
			return
		}

		// Check if smart search is enabled
		if c.Query("smart") != "true" {
			c.Next()
			return
		}

		// TODO: Convert text query to vector using embedding model
		// For now, pass through to regular search
		c.Set("search_mode", "smart")
		c.Set("search_query", query)

		c.Next()
	}
}

// VectorLogger logs vector operations
func VectorLogger() HandlerFunc {
	return func(c *Context) {
		start := time.Now()

		c.Next()

		// Log vector operations
		if c.Request.URL.Path == "/api/vectors/search" {
			duration := time.Since(start)
			fmt.Printf("[Vector] Search completed in %v\n", duration)
		}
	}
}

// Normalize normalizes a vector to unit length
func Normalize(vector Vector) Vector {
	var norm float32
	for _, val := range vector {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm == 0 {
		return vector
	}

	normalized := make(Vector, len(vector))
	for i, val := range vector {
		normalized[i] = val / norm
	}

	return normalized
}

// VectorToJSON converts a vector to JSON string
func VectorToJSON(vector Vector) string {
	bytes, _ := json.Marshal(vector)
	return string(bytes)
}

// JSONToVector converts JSON string to vector
func JSONToVector(jsonStr string) (Vector, error) {
	var vector Vector
	err := json.Unmarshal([]byte(jsonStr), &vector)
	return vector, err
}
