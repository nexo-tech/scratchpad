package models

import "time"

// NoteView represents a note for template rendering
type NoteView struct {
	ID        string
	Category  string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CategoryView represents a category for template rendering
type CategoryView struct {
	Name     string
	Count    int64
	LastNote time.Time
}
