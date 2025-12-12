package goTap

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoClient wraps mongo.Client for middleware use
type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
	ctx      context.Context
}

// NewMongoClient creates a new MongoDB client wrapper
func NewMongoClient(uri, database string) (*MongoClient, error) {
	ctx := context.Background()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("mongodb connection failed: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongodb ping failed: %w", err)
	}

	return &MongoClient{
		Client:   client,
		Database: client.Database(database),
		ctx:      ctx,
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoClient) Close() error {
	return m.Client.Disconnect(m.ctx)
}

// Collection returns a collection from the database
func (m *MongoClient) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// MongoInject injects MongoDB client into context for use in handlers
func MongoInject(client *MongoClient) HandlerFunc {
	return func(c *Context) {
		c.Set("mongodb", client)
		c.Next()
	}
}

// GetMongo retrieves MongoDB client from context
func GetMongo(c *Context) (*MongoClient, bool) {
	client, exists := c.Get("mongodb")
	if !exists {
		return nil, false
	}
	mongoClient, ok := client.(*MongoClient)
	return mongoClient, ok
}

// MustGetMongo retrieves MongoDB client from context or panics
func MustGetMongo(c *Context) *MongoClient {
	client, ok := GetMongo(c)
	if !ok {
		panic("MongoDB client not found in context")
	}
	return client
}

// MongoHealthCheck returns middleware that checks MongoDB health
func MongoHealthCheck(client *MongoClient) HandlerFunc {
	return func(c *Context) {
		if client == nil || client.Client == nil {
			c.JSON(503, H{
				"status":  "unhealthy",
				"mongodb": "not configured",
			})
			c.Abort()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := client.Client.Ping(ctx, nil); err != nil {
			c.JSON(503, H{
				"status":  "unhealthy",
				"mongodb": "connection failed",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		c.JSON(200, H{
			"status":  "healthy",
			"mongodb": "connected",
		})
	}
}

// MongoLogger middleware logs all MongoDB operations
func MongoLogger() HandlerFunc {
	return func(c *Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log MongoDB operations from context
		if ops, exists := c.Get("mongo_ops"); exists {
			duration := time.Since(start)
			if opsList, ok := ops.([]string); ok {
				for _, op := range opsList {
					fmt.Printf("[MongoDB] %s - %s (%v)\n", c.Request.Method, op, duration)
				}
			}
		}
	}
}

// MongoTransaction wraps a handler in a MongoDB transaction
func MongoTransaction(client *MongoClient) HandlerFunc {
	return func(c *Context) {
		// Start a session
		session, err := client.Client.StartSession()
		if err != nil {
			c.JSON(500, H{"error": "Failed to start MongoDB session"})
			c.Abort()
			return
		}
		defer session.EndSession(context.Background())

		// Start a transaction
		ctx := context.Background()
		err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
			// Begin transaction
			if err := session.StartTransaction(); err != nil {
				return err
			}

			// Store session in context
			c.Set("mongo_session", sc)

			// Process request
			c.Next()

			// Check if there were errors
			if len(c.Errors) > 0 {
				session.AbortTransaction(sc)
				return fmt.Errorf("transaction aborted due to errors")
			}

			// Commit transaction
			return session.CommitTransaction(sc)
		})

		if err != nil {
			c.JSON(500, H{"error": "Transaction failed", "details": err.Error()})
		}
	}
}

// MongoRepository provides common database operations
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository creates a new repository for a collection
func NewMongoRepository(client *MongoClient, collectionName string) *MongoRepository {
	return &MongoRepository{
		collection: client.Collection(collectionName),
	}
}

// FindOne finds a single document by filter
func (r *MongoRepository) FindOne(ctx context.Context, filter interface{}) (*mongo.SingleResult, error) {
	return r.collection.FindOne(ctx, filter), nil
}

// Find finds multiple documents by filter
func (r *MongoRepository) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return r.collection.Find(ctx, filter, opts...)
}

// FindByID finds a document by ID
func (r *MongoRepository) FindByID(ctx context.Context, id interface{}) *mongo.SingleResult {
	return r.collection.FindOne(ctx, bson.M{"_id": id})
}

// InsertOne inserts a single document
func (r *MongoRepository) InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	return r.collection.InsertOne(ctx, document)
}

// InsertMany inserts multiple documents
func (r *MongoRepository) InsertMany(ctx context.Context, documents []interface{}) (*mongo.InsertManyResult, error) {
	return r.collection.InsertMany(ctx, documents)
}

// UpdateOne updates a single document
func (r *MongoRepository) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	return r.collection.UpdateOne(ctx, filter, update)
}

// UpdateByID updates a document by ID
func (r *MongoRepository) UpdateByID(ctx context.Context, id interface{}, update interface{}) (*mongo.UpdateResult, error) {
	return r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
}

// DeleteOne deletes a single document
func (r *MongoRepository) DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
	return r.collection.DeleteOne(ctx, filter)
}

// DeleteByID deletes a document by ID
func (r *MongoRepository) DeleteByID(ctx context.Context, id interface{}) (*mongo.DeleteResult, error) {
	return r.collection.DeleteOne(ctx, bson.M{"_id": id})
}

// CountDocuments counts documents matching filter
func (r *MongoRepository) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return r.collection.CountDocuments(ctx, filter)
}

// Aggregate performs aggregation pipeline
func (r *MongoRepository) Aggregate(ctx context.Context, pipeline interface{}) (*mongo.Cursor, error) {
	return r.collection.Aggregate(ctx, pipeline)
}

// CreateIndex creates an index on the collection
func (r *MongoRepository) CreateIndex(ctx context.Context, keys interface{}, unique bool) (string, error) {
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetUnique(unique),
	}
	return r.collection.Indexes().CreateOne(ctx, indexModel)
}

// MongoCache provides a simple cache layer using MongoDB
type MongoCache struct {
	collection *mongo.Collection
	ttl        time.Duration
}

// NewMongoCache creates a new MongoDB-based cache
func NewMongoCache(client *MongoClient, collectionName string, ttl time.Duration) *MongoCache {
	cache := &MongoCache{
		collection: client.Collection(collectionName),
		ttl:        ttl,
	}

	// Create TTL index for automatic expiration
	ctx := context.Background()
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "expireAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	cache.collection.Indexes().CreateOne(ctx, indexModel)

	return cache
}

// Set stores a value in cache
func (mc *MongoCache) Set(ctx context.Context, key string, value interface{}) error {
	doc := bson.M{
		"_id":      key,
		"value":    value,
		"expireAt": time.Now().Add(mc.ttl),
	}

	opts := options.Replace().SetUpsert(true)
	_, err := mc.collection.ReplaceOne(ctx, bson.M{"_id": key}, doc, opts)
	return err
}

// Get retrieves a value from cache
func (mc *MongoCache) Get(ctx context.Context, key string) (interface{}, error) {
	var result struct {
		Value interface{} `bson:"value"`
	}

	err := mc.collection.FindOne(ctx, bson.M{"_id": key}).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// Delete removes a value from cache
func (mc *MongoCache) Delete(ctx context.Context, key string) error {
	_, err := mc.collection.DeleteOne(ctx, bson.M{"_id": key})
	return err
}

// Clear removes all cache entries
func (mc *MongoCache) Clear(ctx context.Context) error {
	_, err := mc.collection.DeleteMany(ctx, bson.M{})
	return err
}

// MongoAuditLog middleware logs all requests to MongoDB
type MongoAuditLog struct {
	collection  *mongo.Collection
	includeBody bool
}

// NewMongoAuditLog creates a new audit log middleware
func NewMongoAuditLog(client *MongoClient, collectionName string, includeBody bool) *MongoAuditLog {
	return &MongoAuditLog{
		collection:  client.Collection(collectionName),
		includeBody: includeBody,
	}
}

// Middleware returns the audit log middleware
func (mal *MongoAuditLog) Middleware() HandlerFunc {
	return func(c *Context) {
		start := time.Now()

		// Capture request body if needed
		var body []byte
		if mal.includeBody && c.Request.Body != nil {
			body, _ = c.GetRawData()
		}

		// Process request
		c.Next()

		// Create audit log entry
		logEntry := bson.M{
			"timestamp": time.Now(),
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"ip":        c.ClientIP(),
			"status":    c.Writer.Status(),
			"duration":  time.Since(start).Milliseconds(),
			"userAgent": c.Request.UserAgent(),
		}

		if mal.includeBody && len(body) > 0 {
			logEntry["body"] = string(body)
		}

		if len(c.Errors) > 0 {
			logEntry["errors"] = c.Errors.String()
		}

		// Store in MongoDB asynchronously
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			mal.collection.InsertOne(ctx, logEntry)
		}()
	}
}

// MongoPagination provides pagination helper
type MongoPagination struct {
	Page     int64
	PageSize int64
	Total    int64
	Pages    int64
}

// NewMongoPagination creates pagination from context query params
func NewMongoPagination(c *Context) *MongoPagination {
	page := parseInt64(c.DefaultQuery("page", "1"), 1)
	pageSize := parseInt64(c.DefaultQuery("page_size", "20"), 20)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return &MongoPagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// Skip returns the number of documents to skip
func (p *MongoPagination) Skip() int64 {
	return (p.Page - 1) * p.PageSize
}

// FindOptions returns MongoDB find options with pagination
func (p *MongoPagination) FindOptions() *options.FindOptions {
	return options.Find().
		SetSkip(p.Skip()).
		SetLimit(p.PageSize)
}

// SetTotal sets the total count and calculates pages
func (p *MongoPagination) SetTotal(total int64) {
	p.Total = total
	p.Pages = (total + p.PageSize - 1) / p.PageSize
}

// Response returns pagination metadata for API response
func (p *MongoPagination) Response() H {
	return H{
		"page":      p.Page,
		"page_size": p.PageSize,
		"total":     p.Total,
		"pages":     p.Pages,
	}
}

// Helper function to parse int64 with default
func parseInt64(s string, defaultValue int64) int64 {
	var result int64
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// MongoTextSearch provides full-text search functionality
type MongoTextSearch struct {
	repository *MongoRepository
}

// NewMongoTextSearch creates a new text search instance
func NewMongoTextSearch(client *MongoClient, collectionName string) *MongoTextSearch {
	return &MongoTextSearch{
		repository: NewMongoRepository(client, collectionName),
	}
}

// CreateTextIndex creates a text index on specified fields
func (mts *MongoTextSearch) CreateTextIndex(ctx context.Context, fields ...string) error {
	keys := bson.D{}
	for _, field := range fields {
		keys = append(keys, bson.E{Key: field, Value: "text"})
	}

	_, err := mts.repository.CreateIndex(ctx, keys, false)
	return err
}

// Search performs a text search
func (mts *MongoTextSearch) Search(ctx context.Context, query string, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	filter := bson.M{"$text": bson.M{"$search": query}}
	return mts.repository.Find(ctx, filter, opts...)
}
