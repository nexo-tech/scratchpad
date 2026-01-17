# Scratchpad

A knowledge pipeline application for AI agents. Collect notes by category via REST API and MCP, view them with a web UI.

## Features

- **REST API** - Push/query notes from AI agents (Claude Chrome Extension, etc.)
- **MCP Server** - HTTP transport for AI agents (OpenCode, Claude Desktop) to consume data
- **Web UI** - Read-only HTMX interface with Teenage Engineering inspired theme
- **Full-text Search** - MongoDB text index for searching across notes
- **Categories** - Organize notes by topic (e.g., twitter-analytics, content-ideas)

## Quick Start

```bash
# Install dependencies
make deps

# Run development server (hot reload)
make dev

# Or build and run production
make build
./bin/server
```

Server starts at http://localhost:7521

## API Endpoints

### REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/notes` | Create note `{category, content}` |
| GET | `/api/notes` | List notes (query: `category`, `limit`, `offset`) |
| GET | `/api/notes/search` | Search (query: `q`, `category`, `since`, `until`) |
| GET | `/api/notes/{id}` | Get single note |
| DELETE | `/api/notes/{id}` | Delete note |
| GET | `/api/categories` | List all categories with counts |

### MCP Tools (via `/mcp`)

| Tool | Description |
|------|-------------|
| `list_categories` | List all categories with counts |
| `get_notes` | Get notes by category |
| `search_notes` | Full-text search with date filters |
| `get_recent_notes` | Get recent notes across all categories |
| `get_note` | Get note by ID |

## Example Usage

### Push a note (from Claude Chrome Extension)

```bash
curl -X POST http://localhost:7521/api/notes \
  -H "Content-Type: application/json" \
  -d '{
    "category": "twitter-analytics",
    "content": "## Insight\n\nEngagement rate for threads is 3x higher than single tweets..."
  }'
```

### Search notes

```bash
curl "http://localhost:7521/api/notes/search?q=engagement&category=twitter-analytics&since=2026-01-01"
```

### MCP Configuration (for OpenCode)

Add to your MCP config:

```json
{
  "mcpServers": {
    "scratchpad": {
      "url": "http://localhost:7521/mcp"
    }
  }
}
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGODB_URI` | `mongodb://oracle-vm:27017` | MongoDB connection string |
| `PORT` | `7521` | Server port |

## Deployment

Using smdctl (systemd):

```bash
make build
smdctl run -f smdctl.yml
smdctl status scratchpad
smdctl logs -f scratchpad
```

## Tech Stack

- **Go 1.22+** - Backend
- **MongoDB** - Database with text index
- **Templ** - Type-safe templates
- **HTMX** - Web interactivity
- **Pico CSS** - Styling (Teenage Engineering theme)
- **mcp-go** - MCP server with HTTP transport
- **goldmark** - Markdown rendering
