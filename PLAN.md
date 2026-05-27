# Owl File Manager - Progress

## Completed

### Core
- [x] Go backend scaffold (HTTP server on :3721)
- [x] SQLite database with FTS5, migrations via golang-migrate (001–007)
- [x] Store layer - CRUD + search + processing + intelligence queries
- [x] 24+ API endpoints with handlers
- [x] CORS + request logging middleware
- [x] Frontend scaffold (Preact + Vite + TypeScript)
- [x] Tauri v2 shell configured

### Scanner & Extraction
- [x] Directory scanner (hidden file skipping, watched_dir_id, cascade delete)
- [x] Async scan via goroutine
- [x] Content extractor (text, PDF, DOCX, XLSX, PPTX + many more formats)
- [x] Metadata extractor (text stats, image dimensions, SVG, PDF pages, Office doc props)
- [x] Change detection — compares modified_at to skip unchanged files (in UpsertFile)
- [x] Processing status tracking (unprocessed/queued/processing/processed/stale/failed)
- [x] Recovery of stuck files on startup
- [x] Raw file serving (images, text preview)

### Search & Files
- [x] Unified search across 5 scopes (filenames, content, comments, tags, notes) with scope toggle pills
- [x] File detail page (metadata, viewer, extracted content, comments, tags with source badges + Auto-Tag button)
- [x] File filtering and sorting (extension, processing status; sort by name/ext/size/indexed_at)
- [x] Pagination for file lists (50 per page, server-side)
- [x] Clickable file names in file list and search results
- [x] Failed status tooltip in file list
- [x] Error banner on file detail page
- [x] SQLite WAL mode + busy_timeout + single connection (fixes SQLITE_BUSY)

### Virtual Folders
- [x] Create/list/delete/edit folders
- [x] Add/remove files to/from folders (multi-select file picker dialog)
- [x] View folder contents (detail page with files + notes placeholder)
- [x] Two-column layout: manual folders left, auto folder suggestions right
- [x] Accept/dismiss auto folder suggestions (accept → source='manual', dismiss → delete)
- [x] Source filtering: `GET /virtual-folders?source=manual|auto`

### Tags
- [x] Tag CRUD with source tracking ('auto'|'manual')
- [x] Two-column layout: manual tags left, auto tags sidebar right
- [x] Auto tag cards clickable → tag detail preview before accepting
- [x] Tag detail page with Accept/Dismiss buttons for auto tags
- [x] Auto-Tag All / Accept All / Dismiss All bulk actions in sidebar
- [x] `source` column replaces removed `auto_generated` column (migration 007)

### Intelligence
- [x] Analyzer: TF-IDF corpus builder with progress logging
- [x] Auto-tagger: single-file (threshold check) + bulk (two-pass with minTagFileCount=3)
- [x] Folder suggester: keyword clustering with MinFilesForFolder=3, phase-by-phase progress
- [x] 11 intelligence API endpoints
- [x] Frontend: TagsPage, TagDetailPage, FolderSuggestions sidebar, TagSuggestions sidebar

### Logging
- [x] Structured logging: `slog` + `tint` handler (colored console output, auto TTY detection)
- [x] `ProgressLogger` for iteration tracking (scanner every 500, extractor every 100, etc.)
- [x] `LOG_LEVEL` env var (default: info)
- [x] OTEL stub for future integration
- [x] All `log.Printf` replaced with `slog.*`

### Navigation
- [x] Ingest page (watched dirs + file list — was "Dashboard")
- [x] Files (standalone list)
- [x] Search
- [x] Tags
- [x] Folders

## v2 - Future

### Dashboard
- [ ] Proper dashboard page (stats, recent activity, quick actions)

### Automation
- [ ] Batch operations - select multiple files, bulk tag/extract/delete
- [ ] Auto-scan on startup
- [ ] Directory watching (fsnotify) for real-time file changes
- [ ] Background pipeline coordination — scan → extract → tag → suggest

### Intelligence
- [ ] User feedback loop — confirm/dismiss suggestions, learn from feedback
- [ ] Min file count thresholds configurable via settings (currently hardcoded: tags=3, folders=3)

### Notes & Materialization
- [ ] Notes frontend (backend done)
  - Create/edit/delete notes
  - Attach notes to virtual folders
  - Tag notes
- [ ] Materialization — create folder on disk + move/copy files (backend stub exists)
- [ ] Note materialization — write .md files to disk (backend stub exists)

### Media & Desktop
- [ ] Image metadata extraction outside extraction pipeline
- [ ] EXIF data extraction for JPEG
- [ ] Audio/video metadata extraction
- [ ] Thumbnail generation for images
- [ ] OCR + AI vision for image content understanding
- [ ] Tauri desktop integration - system tray, native file dialogs
- [ ] Projects feature

## Architecture

```
backend/
  cmd/owl/main.go          - entry point
  internal/
    db/                     - SQLite init + migrations
    store/                  - data access layer
    api/
      handler/              - HTTP handlers
      middleware/            - CORS, logging
      router.go             - route registration
    scanner/                - directory walker
    extractor/              - content + metadata extraction
    intelligence/           - TF-IDF analyzer, auto-tagger, folder suggester
    logging/                - slog+tint init, progress logger, OTEL stub

frontend/
  src/
    api.ts                  - API types + fetch wrappers
    hooks/queries.ts        - TanStack Query hooks
    pages/                  - route page components
    components/             - shared components
    app.tsx                 - router + layout
    app.css                 - all styles
```

## Key Decisions
- SQLite with WAL mode, single connection, 5s busy timeout
- Processing status tracks extraction pipeline
- File metadata stored as JSON blob in file_metadata column
- Extractor: 50MB file limit, 500KB text limit, processes in size order
- Comments are 1:1 per file
- `source` column ('auto'|'manual') on tags and virtual_folders — no separate dismissed tracking
- Dismissal = delete for both tags and folders
- Auto-tagging: single-file uses threshold check (≥minTagFileCount-1 other files), bulk uses two-pass (collect → filter → write)
- Folder suggestions: keyword clustering with min files threshold, created as source='auto'
- `ListTagsWithCount(source *string)` and `ListVirtualFolders(source *string)` — optional source filtering
- Scanner skips hidden files/dirs, does NOT filter by extension
- Scans run async via goroutine
- Logging: `log/slog` with `tint` handler; OTEL stub for future
- `LOG_LEVEL` env var controls log level (default: info)
- Min file count thresholds: tags ≥3, folders ≥3 (hardcoded, TODO: settings)
