# Owl File Manager - Progress

## Completed

- [x] Go backend scaffold (HTTP server on :3721)
- [x] SQLite database with FTS5, migrations via golang-migrate
- [x] Store layer - CRUD for all resources
- [x] 22+ API endpoints with handlers
- [x] CORS + request logging middleware
- [x] Frontend scaffold (Preact + Vite + TypeScript)
- [x] Tauri v2 shell configured
- [x] Directory scanner (hidden file skipping, watched_dir_id, cascade delete)
- [x] Content extractor (text, PDF, DOCX, XLSX, PPTX)
- [x] Processing status tracking (unprocessed/queued/processing/processed/stale/failed)
- [x] Recovery of stuck files on startup
- [x] Unified search across 5 scopes (filenames, content, comments, tags, notes)
- [x] File detail page (metadata, viewer, extracted content debug, comments, tags)
- [x] File metadata extraction (text stats, image dimensions, SVG, PDF pages, Office doc props)
- [x] Raw file serving (images, text preview)
- [x] SQLite WAL mode + busy_timeout + single connection (fixes SQLITE_BUSY)
- [x] Failed status tooltip in file list
- [x] Error banner on file detail page
- [x] Clickable file names in file list and search results

## Next: v1 - Virtual Folders

- [ ] Virtual folders - frontend (backend done)
  - Create/list/delete folders
  - Add/remove files to/from folders
  - View folder contents

## v1.1 - Filtering & Pagination

- [ ] File filtering and sorting in file list
  - Filter by extension, status, processing status
  - Sort by name, size, date
- [ ] Pagination for file lists (currently capped at 200)

## v1.5 - Automation & Batch

- [ ] Batch operations - select multiple files, bulk tag/extract/delete
- [ ] Auto-scan on startup
- [ ] Directory watching (fsnotify) for real-time file changes

## v2 - Future

- [ ] Notes - frontend (backend done)
  - Create/edit/delete notes
  - Attach notes to virtual folders
  - Tag notes
- [ ] Image metadata extraction outside extraction pipeline
  - Images aren't in supportedExtensions so never get metadata (dimensions) extracted
  - Need a metadata-only pass for non-text file types
- [ ] EXIF data extraction for JPEG
- [ ] Audio/video metadata extraction
- [ ] Thumbnail generation for images
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
- Processing status tracks extraction pipeline (not content_indexed_at NULL)
- File metadata stored as JSON blob in file_metadata column
- Extractor processes files in size order (smallest first), 50MB limit
- Comments are 1:1 per file
