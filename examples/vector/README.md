# Vector Database Integration for goTap

Complete guide to AI-powered search and recommendations using Vector databases with goTap.

## Overview

Vector databases enable semantic search, product recommendations, and AI-powered features by storing and querying high-dimensional vectors (embeddings). This implementation provides:

- ✅ In-memory vector store (for testing/small datasets)
- ✅ Cosine similarity search
- ✅ Product recommendation engine
- ✅ REST API handlers
- ✅ Context injection pattern
- 🔜 External vector DB integrations (Pinecone, Milvus, Qdrant)

## Quick Start

```go
package main

import (
    "github.com/jaswant99k/gotap"
)

func main() {
    r := goTap.New()
    
    // Create vector store
    vectorStore := goTap.NewInMemoryVectorStore()
    
    // Inject into context
    r.Use(goTap.VectorInject(vectorStore))
    
    // REST API endpoints
    r.POST("/vectors/search", goTap.VectorSearchHandler())
    r.POST("/vectors", goTap.VectorInsertHandler())
    r.DELETE("/vectors", goTap.VectorDeleteHandler())
    r.GET("/vectors/:id", goTap.VectorGetHandler())
    
    r.Run(":8080")
}
```

## Core Concepts

### 1. Vectors (Embeddings)

Vectors are numerical representations of data (text, images, products) in high-dimensional space. Similar items have similar vectors.

```go
// Vector is a slice of float32
type Vector []float32

// Example product vectors (simplified - real embeddings are 384-1536 dimensions)
laptopVector := goTap.Vector{0.8, 0.1, 0.3, 0.5} // Electronics, computing
mouseVector := goTap.Vector{0.7, 0.2, 0.2, 0.4}  // Similar to laptop
shirtVector := goTap.Vector{0.1, 0.8, 0.6, 0.2}  // Clothing, different category
```

### 2. Vector Documents

```go
type VectorDocument struct {
    ID       string                 // Unique identifier
    Vector   Vector                 // Embedding vector
    Metadata map[string]interface{} // Additional data
}

// Example
doc := &goTap.VectorDocument{
    ID:     "product-123",
    Vector: goTap.Vector{0.8, 0.1, 0.3, 0.5},
    Metadata: goTap.H{
        "name": "MacBook Pro",
        "category": "electronics",
        "price": 1999.99,
    },
}
```

### 3. Similarity Search

Find documents with similar vectors using cosine similarity:

```go
// Insert documents
store := goTap.NewInMemoryVectorStore()
store.Insert(ctx, []*goTap.VectorDocument{doc1, doc2, doc3})

// Search for similar items
queryVector := goTap.Vector{0.75, 0.15, 0.25, 0.45}
results, err := store.Search(ctx, queryVector, 5) // Top 5 results

// Results are sorted by similarity (highest first)
for _, result := range results {
    fmt.Printf("ID: %s, Similarity: %.4f\n", 
        result.Document.ID, 
        result.Score)
}
```

## Mathematical Functions

### Cosine Similarity

Measures similarity between vectors (0 = different, 1 = identical):

```go
v1 := goTap.Vector{1.0, 2.0, 3.0}
v2 := goTap.Vector{1.0, 2.0, 3.0}
similarity := goTap.CosineSimilarity(v1, v2)
// similarity = 1.0 (perfect match)

v3 := goTap.Vector{3.0, 4.0, 0.0}
similarity = goTap.CosineSimilarity(v1, v3)
// similarity = ~0.8 (similar)
```

### Euclidean Distance

Measures distance between vectors (0 = identical, higher = more different):

```go
distance := goTap.EuclideanDistance(v1, v2)
// Lower distance = more similar
```

### Vector Normalization

Convert vector to unit length:

```go
normalized := goTap.Normalize(vector)
// Length of normalized vector = 1.0
```

## Product Recommendation System

### Basic Setup

```go
r := goTap.New()
store := goTap.NewInMemoryVectorStore()
recommender := goTap.NewProductRecommender(store)

r.Use(goTap.VectorInject(store))

// Add products with embeddings
r.POST("/products", func(c *goTap.Context) {
    var product struct {
        ID       string    `json:"id"`
        Name     string    `json:"name"`
        Category string    `json:"category"`
        Price    float64   `json:"price"`
        Vector   []float32 `json:"vector"`
    }
    c.BindJSON(&product)
    
    // Get recommender from context
    store := goTap.MustGetVectorStore(c)
    recommender := goTap.NewProductRecommender(store)
    
    // Add product
    embedding := &goTap.ProductEmbedding{
        ProductID: product.ID,
        Name:      product.Name,
        Category:  product.Category,
        Price:     product.Price,
        Vector:    product.Vector,
    }
    
    err := recommender.AddProduct(c.Request.Context(), embedding)
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, goTap.H{"message": "Product added"})
})

// Get similar products
r.GET("/products/:id/similar", func(c *goTap.Context) {
    store := goTap.MustGetVectorStore(c)
    recommender := goTap.NewProductRecommender(store)
    
    similar, err := recommender.GetSimilarProducts(
        c.Request.Context(),
        c.Param("id"),
        5, // Top 5 similar products
    )
    
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, goTap.H{"similar_products": similar})
})
```

## POS System Examples

### 1. "Customers Also Bought" Feature

```go
// When customer views a product, show similar products
r.GET("/products/:id/recommendations", func(c *goTap.Context) {
    store := goTap.MustGetVectorStore(c)
    recommender := goTap.NewProductRecommender(store)
    
    productID := c.Param("id")
    
    // Get similar products
    similar, err := recommender.GetSimilarProducts(
        c.Request.Context(),
        productID,
        10, // Top 10 recommendations
    )
    
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    // Filter by category or price range if needed
    filtered := []goTap.ProductEmbedding{}
    for _, prod := range similar {
        if prod.Price < 1000 { // Budget-friendly recommendations
            filtered = append(filtered, prod)
        }
    }
    
    c.JSON(200, goTap.H{
        "product_id": productID,
        "recommendations": filtered[:5], // Top 5
    })
})
```

### 2. Semantic Product Search

```go
// Search products by description (requires embedding service)
r.POST("/search/semantic", func(c *goTap.Context) {
    var request struct {
        Query string `json:"query"` // "lightweight laptop for students"
    }
    c.BindJSON(&request)
    
    // Convert query to vector (use OpenAI, Sentence-BERT, etc.)
    queryVector := getEmbedding(request.Query)
    
    // Search vector store
    store := goTap.MustGetVectorStore(c)
    results, err := store.Search(c.Request.Context(), queryVector, 20)
    
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    // Extract products
    products := []goTap.H{}
    for _, result := range results {
        products = append(products, goTap.H{
            "id": result.Document.ID,
            "name": result.Document.Metadata["name"],
            "price": result.Document.Metadata["price"],
            "relevance": result.Score,
        })
    }
    
    c.JSON(200, goTap.H{
        "query": request.Query,
        "results": products,
    })
})

// Helper function (integrate with your embedding service)
func getEmbedding(text string) goTap.Vector {
    // Call OpenAI API, Sentence-BERT, or local embedding model
    // Returns 384-1536 dimensional vector
    return goTap.Vector{...}
}
```

### 3. Visual Search (Image-based)

```go
// Find products similar to uploaded image
r.POST("/search/visual", func(c *goTap.Context) {
    // Get uploaded image
    file, _ := c.FormFile("image")
    
    // Convert image to vector (use CLIP, ResNet, etc.)
    imageVector := imageToEmbedding(file)
    
    // Search
    store := goTap.MustGetVectorStore(c)
    results, err := store.Search(c.Request.Context(), imageVector, 10)
    
    if err != nil {
        c.JSON(500, goTap.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, goTap.H{"similar_products": results})
})
```

### 4. Smart Upselling

```go
// Recommend higher-value similar products
r.GET("/cart/upsell", func(c *goTap.Context) {
    var cartItems []string
    c.BindJSON(&cartItems)
    
    store := goTap.MustGetVectorStore(c)
    recommender := goTap.NewProductRecommender(store)
    
    upsells := []goTap.ProductEmbedding{}
    
    for _, itemID := range cartItems {
        similar, _ := recommender.GetSimilarProducts(c.Request.Context(), itemID, 5)
        
        // Filter for higher-priced items
        for _, prod := range similar {
            currentPrice := getProductPrice(itemID)
            if prod.Price > currentPrice && prod.Price < currentPrice*1.5 {
                upsells = append(upsells, prod)
            }
        }
    }
    
    c.JSON(200, goTap.H{"upsell_recommendations": upsells})
})
```

### 5. Personalized Homepage

```go
// Show products similar to user's purchase history
r.GET("/personalized/feed", func(c *goTap.Context) {
    userID := c.Query("user_id")
    
    // Get user's purchase history (from database)
    purchasedProducts := getUserPurchaseHistory(userID)
    
    store := goTap.MustGetVectorStore(c)
    recommender := goTap.NewProductRecommender(store)
    
    recommendations := []goTap.ProductEmbedding{}
    seen := make(map[string]bool)
    
    // Get recommendations based on each purchased item
    for _, productID := range purchasedProducts {
        similar, _ := recommender.GetSimilarProducts(c.Request.Context(), productID, 3)
        
        for _, prod := range similar {
            if !seen[prod.ProductID] && !isPurchased(productID, purchasedProducts) {
                recommendations = append(recommendations, prod)
                seen[prod.ProductID] = true
            }
        }
    }
    
    c.JSON(200, goTap.H{
        "user_id": userID,
        "personalized_feed": recommendations[:20], // Top 20
    })
})
```

## Generating Embeddings

### Option 1: OpenAI Embeddings API

```go
import "github.com/sashabaranov/go-openai"

func generateEmbedding(text string) goTap.Vector {
    client := openai.NewClient("your-api-key")
    
    resp, err := client.CreateEmbeddings(
        context.Background(),
        openai.EmbeddingRequest{
            Model: openai.AdaEmbeddingV2,
            Input: []string{text},
        },
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Convert to goTap.Vector
    embedding := make(goTap.Vector, len(resp.Data[0].Embedding))
    for i, val := range resp.Data[0].Embedding {
        embedding[i] = val
    }
    
    return embedding
}
```

### Option 2: Local Embedding Model (Sentence-BERT)

```python
# Python service
from sentence_transformers import SentenceTransformer
from flask import Flask, request, jsonify

app = Flask(__name__)
model = SentenceTransformer('all-MiniLM-L6-v2')  # 384 dimensions

@app.route('/embed', methods=['POST'])
def embed():
    text = request.json['text']
    embedding = model.encode(text).tolist()
    return jsonify({'embedding': embedding})

if __name__ == '__main__':
    app.run(port=5000)
```

```go
// Call from Go
func callEmbeddingService(text string) goTap.Vector {
    payload := map[string]string{"text": text}
    resp, _ := http.Post("http://localhost:5000/embed", "application/json", payload)
    
    var result struct {
        Embedding []float32 `json:"embedding"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    
    return goTap.Vector(result.Embedding)
}
```

### Option 3: Pre-computed Embeddings

```go
// Generate embeddings offline and store with products
type ProductWithEmbedding struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Embedding   []float32 `json:"embedding"` // Pre-computed
}

// Bulk insert during data import
func importProducts(products []ProductWithEmbedding) {
    store := goTap.NewInMemoryVectorStore()
    ctx := context.Background()
    
    documents := make([]*goTap.VectorDocument, len(products))
    for i, prod := range products {
        documents[i] = &goTap.VectorDocument{
            ID:     prod.ID,
            Vector: goTap.Vector(prod.Embedding),
            Metadata: goTap.H{
                "name": prod.Name,
                "description": prod.Description,
            },
        }
    }
    
    store.Insert(ctx, documents)
}
```

## Advanced Features

### 1. Filtered Search

```go
r.POST("/search/filtered", goTap.VectorSearchHandler())

// Request body
{
    "vector": [0.8, 0.1, 0.3, ...],
    "limit": 10,
    "filters": {
        "category": "electronics",
        "price_range": {"min": 100, "max": 1000}
    }
}
```

### 2. Batch Operations

```go
// Insert many vectors at once
documents := []*goTap.VectorDocument{doc1, doc2, doc3, ...}
store.Insert(ctx, documents)

// Delete many vectors
store.Delete(ctx, []string{"id1", "id2", "id3"})
```

### 3. Vector Logging

```go
r.Use(goTap.VectorLogger())
// Logs all vector operations with timing
```

### 4. JSON Serialization

```go
// Save vectors to file
vectorJSON := goTap.VectorToJSON(vector)
ioutil.WriteFile("vector.json", []byte(vectorJSON), 0644)

// Load vectors from file
data, _ := ioutil.ReadFile("vector.json")
vector, _ := goTap.JSONToVector(string(data))
```

## External Vector Databases

### Pinecone Integration (Coming Soon)

```go
// Planned API
pineconeStore := goTap.NewPineconeStore(apiKey, index)
r.Use(goTap.VectorInject(pineconeStore))
```

### Milvus Integration (Coming Soon)

```go
// Planned API
milvusStore := goTap.NewMilvusStore("localhost:19530", collection)
r.Use(goTap.VectorInject(milvusStore))
```

### Qdrant Integration (Coming Soon)

```go
// Planned API
qdrantStore := goTap.NewQdrantStore("localhost:6333", collection)
r.Use(goTap.VectorInject(qdrantStore))
```

## Performance Tips

1. **Vector Dimensions**: 
   - 384 dimensions: Fast, good quality (Sentence-BERT)
   - 768 dimensions: Better quality (BERT)
   - 1536 dimensions: Best quality (OpenAI ada-002)

2. **Normalization**: Normalize vectors for faster cosine similarity

3. **Batch Inserts**: Insert multiple vectors at once

4. **Caching**: Cache frequently accessed vectors

5. **Index Size**: Monitor memory usage for large datasets

## Testing

```go
func TestVectorSearch(t *testing.T) {
    store := goTap.NewInMemoryVectorStore()
    ctx := context.Background()
    
    // Insert test data
    documents := []*goTap.VectorDocument{
        {ID: "1", Vector: goTap.Vector{1.0, 0.0, 0.0}},
        {ID: "2", Vector: goTap.Vector{0.9, 0.1, 0.0}},
        {ID: "3", Vector: goTap.Vector{0.0, 1.0, 0.0}},
    }
    store.Insert(ctx, documents)
    
    // Search
    results, err := store.Search(ctx, goTap.Vector{1.0, 0.0, 0.0}, 2)
    if err != nil {
        t.Fatal(err)
    }
    
    // Verify
    if len(results) != 2 {
        t.Errorf("Expected 2 results, got %d", len(results))
    }
    
    if results[0].Document.ID != "1" {
        t.Error("Expected ID '1' as top result")
    }
}
```

## Deployment

### Small Datasets (< 1M vectors)
- Use `InMemoryVectorStore` (included)
- Fast, no dependencies
- Suitable for prototyping and small stores

### Large Datasets (> 1M vectors)
- Use external vector database (Pinecone, Milvus, Qdrant)
- Horizontal scaling
- Advanced features (filtering, hybrid search)

## Real-World Example

Complete POS system with vector search:

```go
package main

import (
    "github.com/jaswant99k/gotap"
)

func main() {
    r := goTap.New()
    
    // Setup stores
    vectorStore := goTap.NewInMemoryVectorStore()
    recommender := goTap.NewProductRecommender(vectorStore)
    
    // Middleware
    r.Use(goTap.VectorInject(vectorStore))
    r.Use(goTap.VectorLogger())
    
    // Load products with embeddings from database
    loadProducts(vectorStore)
    
    // API routes
    api := r.Group("/api/v1")
    {
        // Product recommendations
        api.GET("/products/:id/similar", getSimilarProducts(recommender))
        api.GET("/products/:id/upsell", getUpsellProducts(recommender))
        
        // Search
        api.POST("/search/semantic", semanticSearch)
        api.POST("/search/visual", visualSearch)
        
        // Personalization
        api.GET("/personalized/feed", personalizedFeed(recommender))
        api.GET("/cart/recommendations", cartRecommendations(recommender))
    }
    
    r.Run(":8080")
}

func loadProducts(store goTap.VectorStore) {
    // Load from database and insert into vector store
    // This would typically be done during application startup
}
```

## Learn More

- [Understanding Vector Embeddings](https://platform.openai.com/docs/guides/embeddings)
- [Sentence-BERT](https://www.sbert.net/)
- [CLIP for Image Embeddings](https://github.com/openai/CLIP)
- [Vector Database Comparison](https://benchmark.vectorview.ai/)
