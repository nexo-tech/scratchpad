package notes

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	repo *Repo
	md   goldmark.Markdown
}

func NewService(repo *Repo) *Service {
	return &Service{
		repo: repo,
		md:   goldmark.New(),
	}
}

// Create creates a new note
func (s *Service) Create(ctx context.Context, input CreateNoteInput) (*Note, error) {
	// Normalize category: lowercase, replace spaces with hyphens
	category := strings.ToLower(strings.TrimSpace(input.Category))
	category = strings.ReplaceAll(category, " ", "-")

	if category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}

	note := &Note{
		Category: category,
		Content:  input.Content,
	}

	if err := s.repo.Insert(ctx, note); err != nil {
		return nil, err
	}

	return note, nil
}

// GetByID retrieves a note by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Note, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid note ID: %w", err)
	}
	return s.repo.FindByID(ctx, oid)
}

// List retrieves notes with optional filters
func (s *Service) List(ctx context.Context, q ListQuery) ([]*Note, error) {
	return s.repo.List(ctx, q)
}

// Search performs full-text search
func (s *Service) Search(ctx context.Context, q SearchQuery) ([]*Note, error) {
	return s.repo.Search(ctx, q)
}

// GetRecent retrieves most recent notes
func (s *Service) GetRecent(ctx context.Context, q SearchQuery) ([]*Note, error) {
	return s.repo.GetRecent(ctx, q.Limit, q.Since)
}

// ListCategories returns all categories with stats
func (s *Service) ListCategories(ctx context.Context) ([]*Category, error) {
	return s.repo.ListCategories(ctx)
}

// Delete removes a note by ID
func (s *Service) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid note ID: %w", err)
	}
	return s.repo.Delete(ctx, oid)
}

// RenderMarkdown converts markdown content to HTML
func (s *Service) RenderMarkdown(content string) string {
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(content), &buf); err != nil {
		return content // Return raw content on error
	}
	return buf.String()
}

// Count returns total note count
func (s *Service) Count(ctx context.Context, category string) (int64, error) {
	return s.repo.Count(ctx, category)
}
