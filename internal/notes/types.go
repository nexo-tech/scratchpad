package notes

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Note represents a scratchpad note with category
type Note struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Category  string             `bson:"category" json:"category"`
	Content   string             `bson:"content" json:"content"` // markdown
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`
}

// Category represents aggregated category info
type Category struct {
	Name     string    `bson:"_id" json:"name"`
	Count    int64     `bson:"count" json:"count"`
	LastNote time.Time `bson:"last_note" json:"lastNote"`
}

// CreateNoteInput is the input for creating a note
type CreateNoteInput struct {
	Category string `json:"category"`
	Content  string `json:"content"`
}

// SearchQuery represents search parameters
type SearchQuery struct {
	Query    string     // full-text search query
	Category string     // filter by category
	Since    *time.Time // notes after this date
	Until    *time.Time // notes before this date
	Limit    int
	Offset   int
}

// ListQuery represents list parameters
type ListQuery struct {
	Category string
	Limit    int
	Offset   int
}
