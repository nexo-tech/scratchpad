package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"scratchpad/internal/notes"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates an MCP server with tools for scratchpad operations
func NewServer(svc *notes.Service) *server.MCPServer {
	s := server.NewMCPServer(
		"Scratchpad",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Tool: list_categories - List all categories with counts
	s.AddTool(
		mcp.NewTool("list_categories",
			mcp.WithDescription("List all note categories with counts and last activity date. Use this to understand what topics are available in the scratchpad."),
		),
		handleListCategories(svc),
	)

	// Tool: get_notes - Get notes by category
	s.AddTool(
		mcp.NewTool("get_notes",
			mcp.WithDescription("Get notes from a specific category, ordered by newest first. Use this to retrieve all notes in a topic."),
			mcp.WithString("category",
				mcp.Required(),
				mcp.Description("Category name (e.g., 'twitter-analytics', 'content-ideas')"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of notes to return (default: 50, max: 200)"),
			),
			mcp.WithNumber("offset",
				mcp.Description("Number of notes to skip for pagination (default: 0)"),
			),
		),
		handleGetNotes(svc),
	)

	// Tool: search_notes - Full-text search
	s.AddTool(
		mcp.NewTool("search_notes",
			mcp.WithDescription("Full-text search across notes with optional category and date filtering. Use this to find specific information across all notes or within a category."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query - searches note content"),
			),
			mcp.WithString("category",
				mcp.Description("Optional: Filter by category name"),
			),
			mcp.WithString("since",
				mcp.Description("Optional: Only return notes created after this date (ISO format: YYYY-MM-DD or RFC3339)"),
			),
			mcp.WithString("until",
				mcp.Description("Optional: Only return notes created before this date (ISO format: YYYY-MM-DD or RFC3339)"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of notes to return (default: 50, max: 200)"),
			),
		),
		handleSearchNotes(svc),
	)

	// Tool: get_recent_notes - Get most recent notes across all categories
	s.AddTool(
		mcp.NewTool("get_recent_notes",
			mcp.WithDescription("Get the most recent notes across all categories. Use this to see what's new or to get an overview of recent activity."),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of notes to return (default: 20, max: 100)"),
			),
			mcp.WithString("since",
				mcp.Description("Optional: Only return notes created after this date (ISO format: YYYY-MM-DD or RFC3339)"),
			),
		),
		handleGetRecentNotes(svc),
	)

	// Tool: get_note - Get a specific note by ID
	s.AddTool(
		mcp.NewTool("get_note",
			mcp.WithDescription("Get a specific note by its ID. Use this when you have a note ID and need the full content."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The note ID (24-character hex string)"),
			),
		),
		handleGetNote(svc),
	)

	return s
}

// CategoryResult represents a category with its metadata
type CategoryResult struct {
	Name     string    `json:"name"`
	Count    int64     `json:"count"`
	LastNote time.Time `json:"lastNote"`
}

// NoteResult represents a note in API responses
type NoteResult struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func handleListCategories(svc *notes.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		categories, err := svc.ListCategories(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list categories: %v", err)), nil
		}

		// Convert to result format
		results := make([]CategoryResult, len(categories))
		for i, cat := range categories {
			results[i] = CategoryResult{
				Name:     cat.Name,
				Count:    cat.Count,
				LastNote: cat.LastNote,
			}
		}

		data, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleGetNotes(svc *notes.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		category, err := req.RequireString("category")
		if err != nil {
			return mcp.NewToolResultError("category is required"), nil
		}

		limit := req.GetInt("limit", 50)
		offset := req.GetInt("offset", 0)

		noteList, err := svc.List(ctx, notes.ListQuery{
			Category: category,
			Limit:    limit,
			Offset:   offset,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get notes: %v", err)), nil
		}

		results := notesToResults(noteList)
		data, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleSearchNotes(svc *notes.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("query is required"), nil
		}

		q := notes.SearchQuery{
			Query:    query,
			Category: req.GetString("category", ""),
			Limit:    req.GetInt("limit", 50),
		}

		// Parse since date
		if since := req.GetString("since", ""); since != "" {
			t, err := parseDate(since)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid 'since' date format: %v", err)), nil
			}
			q.Since = &t
		}

		// Parse until date
		if until := req.GetString("until", ""); until != "" {
			t, err := parseDate(until)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid 'until' date format: %v", err)), nil
			}
			q.Until = &t
		}

		noteList, err := svc.Search(ctx, q)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search notes: %v", err)), nil
		}

		results := notesToResults(noteList)
		data, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleGetRecentNotes(svc *notes.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := notes.SearchQuery{
			Limit: req.GetInt("limit", 20),
		}

		// Parse since date
		if since := req.GetString("since", ""); since != "" {
			t, err := parseDate(since)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid 'since' date format: %v", err)), nil
			}
			q.Since = &t
		}

		noteList, err := svc.GetRecent(ctx, q)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get recent notes: %v", err)), nil
		}

		results := notesToResults(noteList)
		data, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func handleGetNote(svc *notes.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError("id is required"), nil
		}

		note, err := svc.GetByID(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get note: %v", err)), nil
		}

		result := NoteResult{
			ID:        note.ID.Hex(),
			Category:  note.Category,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

// Helper functions

func notesToResults(noteList []*notes.Note) []NoteResult {
	results := make([]NoteResult, len(noteList))
	for i, note := range noteList {
		results[i] = NoteResult{
			ID:        note.ID.Hex(),
			Category:  note.Category,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
		}
	}
	return results
}

func parseDate(s string) (time.Time, error) {
	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try YYYY-MM-DD
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected YYYY-MM-DD or RFC3339 format")
}
