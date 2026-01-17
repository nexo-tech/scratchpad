package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"scratchpad/internal/db"
	mcpserver "scratchpad/internal/mcp"
	"scratchpad/internal/notes"

	"github.com/mark3labs/mcp-go/server"
)

//go:embed static
var staticFS embed.FS

func main() {
	// Config
	mongoURI := getEnv("MONGODB_URI", "mongodb://oracle-vm:27017")
	port := getEnv("PORT", "7521")

	// Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Context for startup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	logger.Info("connecting to MongoDB", "uri", mongoURI)
	database, err := db.Connect(ctx, mongoURI, "scratchpad")
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	logger.Info("connected to MongoDB")

	// Wire dependencies
	noteRepo := notes.NewRepo(database)
	if err := noteRepo.EnsureIndexes(ctx); err != nil {
		logger.Warn("failed to ensure indexes", "error", err)
	}
	noteSvc := notes.NewService(noteRepo)
	noteHandler := notes.NewHandler(noteSvc, logger)

	// Create MCP server
	mcpSrv := mcpserver.NewServer(noteSvc)

	// HTTP router
	mux := http.NewServeMux()

	// Static files
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to get static fs: %v", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))

	// REST API endpoints
	mux.HandleFunc("POST /api/notes", noteHandler.CreateNote)
	mux.HandleFunc("GET /api/notes", noteHandler.ListNotes)
	mux.HandleFunc("GET /api/notes/search", noteHandler.SearchNotes)
	mux.HandleFunc("GET /api/notes/{id}", noteHandler.GetNote)
	mux.HandleFunc("DELETE /api/notes/{id}", noteHandler.DeleteNote)
	mux.HandleFunc("GET /api/categories", noteHandler.ListCategories)

	// HTMX Web UI (read-only)
	mux.HandleFunc("GET /", noteHandler.HomePage)
	mux.HandleFunc("GET /category/{name}", noteHandler.CategoryPage)
	mux.HandleFunc("GET /search", noteHandler.SearchPage)
	mux.HandleFunc("GET /fragments/notes", noteHandler.NotesFragment)
	mux.HandleFunc("GET /fragments/search", noteHandler.SearchFragment)

	// MCP endpoint (HTTP transport)
	// MCP uses POST for requests and GET for SSE streams
	mcpHTTP := server.NewStreamableHTTPServer(mcpSrv)
	mux.Handle("POST /mcp", mcpHTTP)
	mux.Handle("GET /mcp", mcpHTTP)
	mux.Handle("DELETE /mcp", mcpHTTP)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Start server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		logger.Info("shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown error", "error", err)
		}
	}()

	logger.Info("server starting", "port", port)
	logger.Info("endpoints available",
		"web", "http://localhost:"+port,
		"api", "http://localhost:"+port+"/api",
		"mcp", "http://localhost:"+port+"/mcp",
	)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}

	logger.Info("server stopped")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
