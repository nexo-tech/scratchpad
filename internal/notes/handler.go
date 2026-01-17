package notes

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"scratchpad/views/components"
	"scratchpad/views/models"
	"scratchpad/views/pages"
)

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// --- REST API Handlers ---

// CreateNote handles POST /api/notes
func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	var input CreateNoteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	note, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.log.Error("failed to create note", "error", err)
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.jsonResponse(w, note, http.StatusCreated)
}

// GetNote handles GET /api/notes/{id}
func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.jsonError(w, "note ID required", http.StatusBadRequest)
		return
	}

	note, err := h.svc.GetByID(r.Context(), id)
	if errors.Is(err, ErrNoteNotFound) {
		h.jsonError(w, "note not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("failed to get note", "error", err)
		h.jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, note, http.StatusOK)
}

// ListNotes handles GET /api/notes
func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	q := ListQuery{
		Category: r.URL.Query().Get("category"),
		Limit:    h.parseInt(r.URL.Query().Get("limit"), 50),
		Offset:   h.parseInt(r.URL.Query().Get("offset"), 0),
	}

	notes, err := h.svc.List(r.Context(), q)
	if err != nil {
		h.log.Error("failed to list notes", "error", err)
		h.jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, notes, http.StatusOK)
}

// SearchNotes handles GET /api/notes/search
func (h *Handler) SearchNotes(w http.ResponseWriter, r *http.Request) {
	q := SearchQuery{
		Query:    r.URL.Query().Get("q"),
		Category: r.URL.Query().Get("category"),
		Limit:    h.parseInt(r.URL.Query().Get("limit"), 50),
		Offset:   h.parseInt(r.URL.Query().Get("offset"), 0),
	}

	// Parse date filters
	if since := r.URL.Query().Get("since"); since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			t, err = time.Parse("2006-01-02", since)
		}
		if err == nil {
			q.Since = &t
		}
	}
	if until := r.URL.Query().Get("until"); until != "" {
		t, err := time.Parse(time.RFC3339, until)
		if err != nil {
			t, err = time.Parse("2006-01-02", until)
		}
		if err == nil {
			q.Until = &t
		}
	}

	notes, err := h.svc.Search(r.Context(), q)
	if err != nil {
		h.log.Error("failed to search notes", "error", err)
		h.jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, notes, http.StatusOK)
}

// ListCategories handles GET /api/categories
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		h.log.Error("failed to list categories", "error", err)
		h.jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, categories, http.StatusOK)
}

// DeleteNote handles DELETE /api/notes/{id}
func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.jsonError(w, "note ID required", http.StatusBadRequest)
		return
	}

	err := h.svc.Delete(r.Context(), id)
	if errors.Is(err, ErrNoteNotFound) {
		h.jsonError(w, "note not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("failed to delete note", "error", err)
		h.jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Helper methods ---

func (h *Handler) jsonResponse(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *Handler) parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// --- View model converters ---

func (h *Handler) categoriesToViews(categories []*Category) []models.CategoryView {
	views := make([]models.CategoryView, len(categories))
	for i, cat := range categories {
		views[i] = models.CategoryView{
			Name:     cat.Name,
			Count:    cat.Count,
			LastNote: cat.LastNote,
		}
	}
	return views
}

func (h *Handler) notesToViews(notes []*Note) []models.NoteView {
	views := make([]models.NoteView, len(notes))
	for i, note := range notes {
		views[i] = models.NoteView{
			ID:        note.ID.Hex(),
			Category:  note.Category,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
		}
	}
	return views
}

// --- HTMX Web Handlers ---

// HomePage handles GET /
func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		h.log.Error("failed to list categories", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	totalNotes, _ := h.svc.Count(r.Context(), "")

	pages.HomePage(h.categoriesToViews(categories), totalNotes).Render(r.Context(), w)
}

// CategoryPage handles GET /category/{name}
func (h *Handler) CategoryPage(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("name")
	if category == "" {
		http.NotFound(w, r)
		return
	}

	noteList, err := h.svc.List(r.Context(), ListQuery{
		Category: category,
		Limit:    50,
	})
	if err != nil {
		h.log.Error("failed to list notes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	totalCount, _ := h.svc.Count(r.Context(), category)

	// Convert to view models and render markdown
	noteViews := h.notesToViews(noteList)
	renderedContent := make(map[string]string)
	for _, note := range noteViews {
		renderedContent[note.ID] = h.svc.RenderMarkdown(note.Content)
	}

	pages.CategoryPage(category, noteViews, totalCount, renderedContent).Render(r.Context(), w)
}

// SearchPage handles GET /search
func (h *Handler) SearchPage(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		h.log.Error("failed to list categories", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	pages.SearchPage(h.categoriesToViews(categories)).Render(r.Context(), w)
}

// NotesFragment handles GET /fragments/notes (HTMX partial)
func (h *Handler) NotesFragment(w http.ResponseWriter, r *http.Request) {
	q := ListQuery{
		Category: r.URL.Query().Get("category"),
		Limit:    h.parseInt(r.URL.Query().Get("limit"), 50),
		Offset:   h.parseInt(r.URL.Query().Get("offset"), 0),
	}

	noteList, err := h.svc.List(r.Context(), q)
	if err != nil {
		h.log.Error("failed to list notes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Convert to view models and render markdown
	noteViews := h.notesToViews(noteList)
	renderedContent := make(map[string]string)
	for _, note := range noteViews {
		renderedContent[note.ID] = h.svc.RenderMarkdown(note.Content)
	}

	components.NoteCardList(noteViews, renderedContent).Render(r.Context(), w)
}

// SearchFragment handles GET /fragments/search (HTMX partial)
func (h *Handler) SearchFragment(w http.ResponseWriter, r *http.Request) {
	q := SearchQuery{
		Query:    r.URL.Query().Get("q"),
		Category: r.URL.Query().Get("category"),
		Limit:    h.parseInt(r.URL.Query().Get("limit"), 50),
	}

	// Parse date filters
	if since := r.URL.Query().Get("since"); since != "" {
		t, err := time.Parse("2006-01-02", since)
		if err == nil {
			q.Since = &t
		}
	}
	if until := r.URL.Query().Get("until"); until != "" {
		t, err := time.Parse("2006-01-02", until)
		if err == nil {
			q.Until = &t
		}
	}

	noteList, err := h.svc.Search(r.Context(), q)
	if err != nil {
		h.log.Error("failed to search notes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Convert to view models and render markdown
	noteViews := h.notesToViews(noteList)
	renderedContent := make(map[string]string)
	for _, note := range noteViews {
		renderedContent[note.ID] = h.svc.RenderMarkdown(note.Content)
	}

	pages.SearchResults(noteViews, renderedContent, q.Query).Render(r.Context(), w)
}
