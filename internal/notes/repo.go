package notes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrNoteNotFound = errors.New("note not found")
)

type Repo struct {
	coll *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{coll: db.Collection("notes")}
}

// EnsureIndexes creates necessary indexes for the notes collection
func (r *Repo) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "content", Value: "text"}},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
	}

	_, err := r.coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}
	return nil
}

// Insert creates a new note
func (r *Repo) Insert(ctx context.Context, n *Note) error {
	n.ID = primitive.NewObjectID()
	n.CreatedAt = time.Now()
	n.UpdatedAt = n.CreatedAt

	_, err := r.coll.InsertOne(ctx, n)
	if err != nil {
		return fmt.Errorf("insert note: %w", err)
	}
	return nil
}

// FindByID retrieves a note by its ID
func (r *Repo) FindByID(ctx context.Context, id primitive.ObjectID) (*Note, error) {
	var note Note
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&note)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find note %s: %w", id, err)
	}
	return &note, nil
}

// List retrieves notes with optional category filter, sorted by created_at desc
func (r *Repo) List(ctx context.Context, q ListQuery) ([]*Note, error) {
	filter := bson.M{}
	if q.Category != "" {
		filter["category"] = q.Category
	}

	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	opts := options.Find().
		SetLimit(int64(q.Limit)).
		SetSkip(int64(q.Offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer cursor.Close(ctx)

	var notes []*Note
	if err := cursor.All(ctx, &notes); err != nil {
		return nil, fmt.Errorf("decode notes: %w", err)
	}
	return notes, nil
}

// Search performs full-text search with optional filters
func (r *Repo) Search(ctx context.Context, q SearchQuery) ([]*Note, error) {
	filter := bson.M{}

	// Full-text search
	if q.Query != "" {
		filter["$text"] = bson.M{"$search": q.Query}
	}

	// Category filter
	if q.Category != "" {
		filter["category"] = q.Category
	}

	// Date range filter
	if q.Since != nil || q.Until != nil {
		dateFilter := bson.M{}
		if q.Since != nil {
			dateFilter["$gte"] = *q.Since
		}
		if q.Until != nil {
			dateFilter["$lte"] = *q.Until
		}
		filter["created_at"] = dateFilter
	}

	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	opts := options.Find().
		SetLimit(int64(q.Limit)).
		SetSkip(int64(q.Offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	// Add text score for relevance sorting when doing text search
	if q.Query != "" {
		opts.SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}})
		opts.SetSort(bson.D{
			{Key: "score", Value: bson.M{"$meta": "textScore"}},
			{Key: "created_at", Value: -1},
		})
	}

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer cursor.Close(ctx)

	var notes []*Note
	if err := cursor.All(ctx, &notes); err != nil {
		return nil, fmt.Errorf("decode search results: %w", err)
	}
	return notes, nil
}

// GetRecent retrieves most recent notes across all categories
func (r *Repo) GetRecent(ctx context.Context, limit int, since *time.Time) ([]*Note, error) {
	filter := bson.M{}
	if since != nil {
		filter["created_at"] = bson.M{"$gte": *since}
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("get recent notes: %w", err)
	}
	defer cursor.Close(ctx)

	var notes []*Note
	if err := cursor.All(ctx, &notes); err != nil {
		return nil, fmt.Errorf("decode recent notes: %w", err)
	}
	return notes, nil
}

// ListCategories returns all categories with counts and last note time
func (r *Repo) ListCategories(ctx context.Context) ([]*Category, error) {
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":       "$category",
				"count":     bson.M{"$sum": 1},
				"last_note": bson.M{"$max": "$created_at"},
			},
		},
		{
			"$sort": bson.M{"last_note": -1},
		},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate categories: %w", err)
	}
	defer cursor.Close(ctx)

	var categories []*Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, fmt.Errorf("decode categories: %w", err)
	}
	return categories, nil
}

// Delete removes a note by ID
func (r *Repo) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNoteNotFound
	}
	return nil
}

// Count returns the total number of notes, optionally filtered by category
func (r *Repo) Count(ctx context.Context, category string) (int64, error) {
	filter := bson.M{}
	if category != "" {
		filter["category"] = category
	}
	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count notes: %w", err)
	}
	return count, nil
}
