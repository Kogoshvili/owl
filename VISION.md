# Owl

A desktop application for automatic file organization. Instead of manually sorting files across Downloads, Desktop, and Documents, Owl analyzes your content and creates an intelligent interface to access everything structurally.

## Core Identity

Owl is a **folder suggestion engine**. It watches your messy download directories, analyzes file content and names, and suggests how to group related files into folders. You review the suggestions, refine names with LLM help, and materialize them to disk.

Owl does **one thing**: take a pile of unorganized files and turn them into organized folders. It is not a file manager, not a search tool, not a notes app, not a tagging system.

## Scope

### What's In

| Feature | Status |
|---------|--------|
| Watched directories | ✓ Add folders to watch, scan to index |
| File scanning | ✓ Walk directories recursively, record metadata |
| Folder guarding | ✓ LLM decides which folders are already organized (skip) vs. messy (process) |
| Content extraction | ✓ Extract text from PDF, DOCX, code, etc. for keyword analysis |
| Keyword extraction | ✓ TF-IDF or embedding-based; fallback to filename + extension |
| Folder suggestions | ✓ Cluster orphans, name clusters, save to DB |
| LLM refinement | ✓ Rename/describe suggestions on user click |
| Materialization | ✓ Accept suggestion → choose destination → move files to new folder on disk |
| File browser | ✓ Browse folder tree, view files per folder, see guard status |
| Comments | ✓ Per-file notes for context |

### What's Out

| Feature | Why |
|---------|-----|
| Search | Extraction is keyword-focused for suggestion logic, not general-purpose search |
| Tags | No clear user action — keywords are internal to the suggestion pipeline |
| Notes | Different product. `.md` files in a watched dir flow through the pipeline naturally |
| General file management | No rename, delete, move outside of materialization |

## Pipeline

```
Add dirs → Scan → Guard → Extract → Suggest → Review → Materialize
```

Each step is a manual trigger (except scan which auto-runs on add):

1. **Add / Scan** — Register a directory, walk it recursively, record all files
2. **Guard** — LLM classifies folders as organized (skip) or messy (process); max depth 3; user can toggle
3. **Extract** — Extract text content from unguarded orphan files for keyword analysis
4. **Suggest** — Build TF-IDF corpus, cluster orphans via content_tfidf or embeddings strategy, save suggestions to DB
5. **Review** — View suggestions, refine names with LLM, remove individual files
6. **Materialize** — Pick a suggestion, choose destination path (default `~/Owl-organized/`), move files to new folder

### Pipeline States

```
Guarded folders:     🔒 Files belong together — skip entirely
Open folders:        🔓 Files are fair game — may contribute orphans
Orphans:             Files from open folders that don't fit their current location
Extractable orphans: Orphans with supported extensions — queued for content extraction
Suggested orphans:   Assigned to a folder suggestion
Materialized:       Moved to a new folder on disk
```

### Strategies

Two organization backends, configured via `config.json`:

- **`content_tfidf`** (default): TF-IDF vectors + pairwise cosine similarity + flood-fill clustering. Fast (~30s for 12K files).
- **`embeddings`**: LM Studio embedding vectors + DBSCAN clustering. Semantic understanding but slower (~20-40min first run, cached thereafter).

## Materialization Flow

1. Click **Accept** on a suggestion card
2. Enter a destination base path (defaults to `~/Owl-organized/`)
3. Owl creates `{destination}/{suggestion-name}/` and moves all files via `os.Rename`
4. Name collisions handled automatically (`file_1.ext`, `file_2.ext`, ...)
5. Suggestion marked as materialized in DB with green "Materialized" badge
6. Failed moves reported per-file (permissions, not-found)

## Folder Guard

- LLM classifies each folder (BFS top-down, max depth 3)
- Folders with related files → guarded (🔒) — entire subtree skipped
- Messy folders → open (🔓) — children processed
- Deeper than depth 3 → auto-guarded
- User can toggle any folder (saved as `source=user`, always respected)
- Escalation: if folder has 0 files and all children guarded, parent auto-guards

## Use Case

You have PDFs, documents, images, and installers scattered across Downloads and Desktop. Owl will:
- Scan your watched directories and index every file
- Extract text content from supported formats (PDF, DOCX, code, etc.)
- Classify folders as coherent (well-organized — leave alone) or needing attention
- Run analysis strategies on orphan files and suggest folder groupings
- Let you accept, dismiss, or refine suggestions with LLM-assisted naming
- Materialize accepted suggestions to disk

**Example:** You've downloaded research papers, meeting notes, and installer files over months. Owl identifies groups like "Q4 Financial Reports," "Python Project Assets," and "Meeting Notes 2024" — then you accept and they're organized on disk.

## Stack

- **Backend:** Go 1.26 + SQLite with FTS5 (for keyword index, not user search) + stdlib `http.ServeMux` on `:3721`
- **Frontend:** Preact + Vite + TypeScript (works standalone, Tauri v2 shell optional)
- **LLM:** LM Studio (OpenAI-compatible) for folder guard + refinement — default model: `deepseek-r1-distill-qwen-1.5b`
- **Embeddings:** LM Studio `/v1/embeddings` for semantic strategy
- **Database:** `data/owl.db` (project root), migrations via `golang-migrate` with `embed.FS`
- **Config:** `config.json` or env vars (`LLM_ENABLED`, `LLM_BASE_URL`, `LLM_MODEL`, `EMBED_MODEL`, `LOG_LEVEL`)

## Configuration

```json
{
  "llm": {
    "base_url": "http://localhost:1234/v1",
    "model": "deepseek-r1-distill-qwen-1.5b",
    "embed_model": "",
    "folder_strategy": "content_tfidf"
  },
  "log_level": "info"
}
```

Key settings:
| Key | Default | Description |
|-----|---------|-------------|
| `llm.base_url` | `http://localhost:1234/v1` | LM Studio server URL |
| `llm.model` | `deepseek-r1-distill-qwen-1.5b` | LLM for guard + refinement |
| `llm.embed_model` | `""` | Embedding model name (required for embeddings strategy) |
| `llm.folder_strategy` | `"content_tfidf"` | `"content_tfidf"` or `"embeddings"` |

## API

REST API on port 3721. Key endpoints:

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/health` | Health check |
| `GET/POST` | `/watched-directories` | List / add watched dirs |
| `POST` | `/watched-directories/{id}/scan` | Scan a directory |
| `GET/POST` | `/suggestions` | List / create suggestions |
| `GET/PATCH/DELETE` | `/suggestions/{id}` | Get / update / delete a suggestion |
| `POST` | `/suggestions/{id}/materialize` | Accept and materialize to disk |
| `POST` | `/suggestions/{id}/files` | Add files to a suggestion |
| `DELETE` | `/suggestions/{id}/files/{fileId}` | Remove file from suggestion |
| `POST` | `/intelligence/guard/run` | Run folder guard classification |
| `POST` | `/intelligence/files/extract-orphans` | Extract orphan file content |
| `POST` | `/intelligence/folders/suggestions` | Generate folder suggestions |
| `POST` | `/intelligence/refine/folder/{id}` | LLM refine a suggestion name |

## Startup

1. Install Go 1.26+, Node 24+, pnpm 11+
2. `cd backend && go run ./cmd/owl` (server starts on `:3721`)
3. `cd frontend && pnpm dev` (UI at configured port, proxied to `:3721`)
4. Optionally: run LM Studio with an OpenAI-compatible model loaded

## Project Structure

```
owl/
├── VISION.md              # This file
├── backend/
│   ├── cmd/owl/           # Main binary
│   ├── internal/
│   │   ├── api/           # HTTP routes + handlers
│   │   ├── cluster/       # DBSCAN clustering
│   │   ├── config/        # Configuration
│   │   ├── db/            # Migration runner + SQL files
│   │   ├── embedding/     # LM Studio embedding client
│   │   ├── extractor/     # File content extraction
│   │   ├── intelligence/  # TF-IDF, strategies, coherence, suggester
│   │   ├── llm/           # LM Studio LLM client
│   │   ├── logging/       # Tint-based structured logging
│   │   ├── scanner/       # Filesystem walker
│   │   ├── store/         # SQLite queries (all CRUD)
│   │   └── vector/        # Cosine similarity, vector math
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/    # Reusable UI (FileTree, WatchedDirs, etc.)
│   │   ├── hooks/         # TanStack Query hooks
│   │   ├── pages/         # Page components
│   │   └── api.ts         # REST client
│   └── package.json
├── data/                  # Runtime data (DB, logs) — gitignored
└── config.json            # User configuration
```

## Future (Path B)

If the suggestion engine proves useful, potential expansions:

- **Real file manager** — full extraction (PDF OCR, DOCX, images), rich UI, file operations
- **Notes** — dedicated notes data model with the same suggestion pipeline
- **Automation** — fsnotify watching, auto-scan, auto-extract, auto-suggest
- **Media support** — thumbnails, EXIF, audio/video metadata
- **Desktop integration** — Tauri system tray, native file dialogs
