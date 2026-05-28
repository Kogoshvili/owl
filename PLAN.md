# Owl File Manager — Plan

## Current Status: v1.1 (Folders Overhaul)

### What's Done
- Go backend with SQLite FTS5, HTTP API on :3721
- Preact + Vite + TypeScript frontend, Tauri v2 shell
- Scanner, content extractor (txt/md/pdf/docx/xlsx/pptx + more), metadata extractor
- Unified search across 5 scopes (filenames, content, comments, tags, notes)
- File detail page with metadata, viewers, comments, tags
- Virtual folders: create/edit/delete, accept/dismiss auto-suggestions, materialize
- Tags: auto-tag, accept/dismiss, LLM refinement for names/descriptions
- Intelligence module with strategy interface: content_tfidf and embeddings
- Embedding client (LM Studio / OpenAI-compatible), DBSCAN clustering
- LLM integration via LM Studio for tag/folder refinement
- Structured logging with slog + tint

### Tags — Deferred to v1.2
Tags need a separate redesign. Known issues:
- Too many generic tags produced (extension-based, path segments)
- Tags don't consider existing folder structure
- Need smarter content-aware tagging that respects physical folder boundaries
- Tag strategy should be configurable separately from folder strategy

### Current Focus: Smart Folders
The core problem: the system treats all files equally and clusters blindly, ignoring that many files already live in meaningful subfolders. We need to:
1. Show real filesystem folders in the UI (tree view)
2. Analyze subfolder coherence (are files in a subfolder related?)
3. Focus suggestions on orphan files and incoherent subfolders
4. Suggest adding files to existing folders, not just creating new ones
5. Keep only 2 strategies: content_tfidf (fast) and embeddings (better)

---

## Implementation Phases

### Phase 1: Physical Folder Discovery
- Store: `ListPhysicalFolders(watchedDirID)` → builds tree from `parent_dir` values
- API: `GET /intelligence/folders/physical?watched_dir_id=` → returns folder tree
- API: `GET /intelligence/folders/physical/{path}/files` → files in a folder
- Frontend: tree view component for browsing physical folders

### Phase 2: Subfolder Coherence Analysis
- `AnalyzeFolderCoherence(path)` → avg TF-IDF similarity, outlier detection
- API: `GET /intelligence/folders/physical/{path}/coherence`
- Frontend: coherence indicator per folder (green/yellow/red)

### Phase 3: Smart Suggestion Engine
- Classify folders: coherent (leave alone) vs incoherent (needs help)
- Collect orphans: files in root, outliers from incoherent folders
- Generate typed suggestions:
  - `new_folder` — cluster of orphans → new virtual folder
  - `add_to_folder` — orphans related to existing folder
  - `merge_folders` — adjacent similar subfolders
- API: `POST /intelligence/folders/smart-suggest`

### Phase 4: Config & Strategy Simplification
- Remove: path_tfidf, keyword_llm, hybrid strategies
- Keep: content_tfidf (default, fast), embeddings (better, needs LM Studio embedding model)
- Config: `folder_strategy` field in config.json
- Remove strategy dropdown from UI

### Phase 5: Frontend Overhaul
- Folders page: tree view of physical folders + virtual folders below
- Suggestions sidebar: typed suggestions (new_folder, add_to_folder, merge_folders)
- Remove strategy selectors
- Coherence indicators on physical folders

---

## Key Files

### Backend
- `cmd/owl/main.go` — entry point, server setup
- `internal/config/config.go` — config loading (JSON + env vars)
- `internal/db/migrations/` — schema migrations (001-009)
- `internal/store/` — data access layer (files, tags, virtual_folders, etc.)
- `internal/intelligence/` — strategy interface, content_tfidf, embeddings, coherence
- `internal/embedding/` — LM Studio embedding client (OpenAI-compatible)
- `internal/llm/` — LLM client for refinement
- `internal/api/handler/` — HTTP handlers
- `internal/api/router.go` — route definitions

### Frontend
- `src/api.ts` — API client functions
- `src/hooks/queries.ts` — TanStack Query hooks
- `src/components/` — UI components
- `src/pages/` — page-level components
- `src/app.css` — all styles

### Config
- `config.json` (repo root, gitignored) — local development config
- `~/.config/owl/config.json` — production config path
- Env vars: `LLM_ENABLED`, `LLM_BASE_URL`, `LLM_MODEL`, `EMBED_MODEL`
