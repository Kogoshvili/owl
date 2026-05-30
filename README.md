# Owl Folder Suggester

A desktop application for automatic file organization. Instead of manually sorting files across Downloads, Desktop, and Documents, Owl analyzes your content and creates intelligent folder suggestions to organize everything.

## Core Identity

Owl is a **folder suggestion engine**. It watches your messy download directories, analyzes file content and names, and suggests how to group related files into folders. You review the suggestions, refine names with LLM help, and materialize them to disk.

Owl does **one thing**: take a pile of unorganized files and turn them into organized folders. It is not a file manager, not a search tool, not a notes app, not a tagging system.

## Scope

### What's In

| Feature | Status |
|---------|--------|
| Watched directories | ✓ Add folders to watch, scan to index |
| File scanning | ✓ Walk directories recursively, record metadata |
| Auto Folder Guard | ✓ LLM classifies folders as coherent (skip) or messy (process); user can toggle |
| Content extraction | ✓ Extract text from PDF, DOCX, code, etc. for keyword analysis |
| Keyword extraction | ✓ TF-IDF or embedding-based; fallback to filename + extension tags |
| Folder suggestions | ✓ Cluster orphans, name clusters, save to DB |
| LLM refinement | ✓ Rename/describe suggestions on user click |
| Materialization | ✓ Accept suggestion → choose destination (with Browse) → move files to new folder |
| File tree browser | ✓ Browse folder hierarchy, view guard status, rescan/delete watched dirs |
| Desktop build | ✓ Tauri v2 shell, Go sidecar, MSI/DEB/DMG installers via GitHub Actions |
| Process instructions | ✓ How-it-works banner on home page |

### What's Out

| Feature | Why |
|---------|-----|
| Search | Extraction is keyword-focused for suggestion logic, not general-purpose search |
| Tags | No clear user action — keywords are internal to the suggestion pipeline |
| Notes | Different product. `.md` files in a watched dir flow through the pipeline naturally |
| General file management | No rename, delete, move outside of materialization |

## Pipeline

```
Add dirs → Scan → Auto Guard → Extract → Generate → Review → Materialize
```

Each step is a manual trigger from the pipeline bar:

1. **Add directories** — Type a path or use **Browse** to select folders to organize
2. **Auto Guard** — Automatically checks if a folder is coherent (files belong together). Coherent folders are **skipped** — their content won't be used for suggestions. Incoherent folders are **open** for extraction. Click 🔒/🔓 on any folder to override
3. **Extract** — Reads content from supported file types (`.txt`, `.md`, `.py`, `.pdf`, etc.) to extract keywords, metadata, and structure. Even unsupported files still contribute their filename and extension tags — extraction is extra content that improves similarity analysis for better groupings
4. **Generate** — Creates folder suggestions by clustering related files together
5. **Accept** — Click a suggestion to materialize the folder on disk

### Pipeline States

```
Guarded folders:     🔒 Files belong together — skip entirely
Open folders:        🔓 Files are fair game — may contribute orphans
Orphans:             Files from open folders that don't fit their current location
Extractable orphans: Orphans with supported extensions — queued for content extraction
Suggested orphans:   Assigned to a folder suggestion
Materialized:        Moved to a new folder on disk
```

### Strategies

Two organization backends, configured via `config.json` or the strategy dropdown:

- **`content_tfidf`** (default): TF-IDF vectors + pairwise cosine similarity + flood-fill clustering. Fast (~30s for 12K files), works immediately with no model dependency.
- **`embeddings`**: LM Studio embedding vectors + DBSCAN clustering. Semantic understanding but requires an embedding model endpoint.

## Materialization Flow

1. Click **Accept** on a suggestion card
2. A modal appears with **Browse** button to pick destination (uses native folder picker via Tauri, with browser fallback)
3. Edit folder name if desired
4. Owl creates `{destination}/{folder-name}/` and moves all files via `os.Rename`
5. Name collisions handled automatically (`file_1.ext`, `file_2.ext`, ...)
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

- **Backend:** Go 1.26 + SQLite with FTS5 + stdlib `http.ServeMux` on `127.0.0.1:3721`
- **Frontend:** Preact + Vite + TypeScript + TanStack Query
- **Desktop shell:** Tauri v2 (spawns Go as sidecar, serves frontend natively)
- **LLM:** LM Studio (OpenAI-compatible) for folder guard + refinement — default model: `deepseek-r1-distill-qwen-1.5b`
- **Embeddings:** LM Studio `/v1/embeddings` for semantic strategy
- **Database:** SQLite via `modernc.org/sqlite` (no CGO), migrations via `golang-migrate`
- **Config:** `config.json` or env vars (`LLM_ENABLED`, `LLM_BASE_URL`, `LLM_MODEL`, `EMBED_MODEL`, `LOG_LEVEL`)
- **CLI flags:** `--port` (default `3721`), `--data-dir` (default `data`)
- **Build:** `build.sh` (Linux/macOS) / `build.ps1` (Windows) — builds Go binary + Tauri installer
- **CI/CD:** GitHub Actions (`.github/workflows/release.yml`) — builds all 3 platforms on tag push

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
| `--port` | `3721` | Backend HTTP port |
| `--data-dir` | `data` | Directory for database and logs |

## API

All routes are prefixed with `/api`. Server listens on `127.0.0.1:3721`.

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/health` | Health check |
| `GET/POST` | `/api/watched-directories` | List / add watched dirs |
| `POST` | `/api/watched-directories/{id}/scan` | Scan a directory |
| `GET/POST` | `/api/suggestions` | List / create suggestions |
| `GET/PATCH/DELETE` | `/api/suggestions/{id}` | Get / update / delete a suggestion |
| `POST` | `/api/suggestions/{id}/materialize` | Accept and materialize to disk |
| `POST` | `/api/intelligence/guard/run` | Run folder guard classification |
| `POST` | `/api/intelligence/files/extract-orphans` | Extract orphan file content |
| `POST` | `/api/intelligence/folders/suggestions` | Generate folder suggestions |
| `POST` | `/api/intelligence/refine/folder/{id}` | LLM refine a suggestion name |

## Startup

### Development (standalone backend + Vite)

```bash
# Terminal 1: Start Go backend
cd backend
go run ./cmd/owl --port 3721 --data-dir data

# Terminal 2: Start frontend dev server
cd frontend
pnpm dev
```

The Vite dev server proxies `/api` → `http://127.0.0.1:3721`.

### Development (Tauri — spawns backend automatically)

```bash
cd frontend
pnpm tauri dev
```

Tauri spawns the Go sidecar with `--port 3721 --data-dir <project>/backend/data` automatically.

### Production build

```bash
# Linux/macOS
./build.sh

# Windows
.\build.ps1
```

Builds Go binary, copies to Tauri binaries directory, runs `pnpm tauri build`. Resulting installers in `frontend/src-tauri/target/release/bundle/`.

### Data directory

- **Development:** `backend/data/` (DB + logs)
- **Installed app:** `<install-dir>/data/` (e.g., `C:\Program Files\Owl\data\`)

## Project Structure

```
owl/
├── README.md                  # This file
├── SHIPPING.md               # Distribution plan
├── build.sh                  # Linux/macOS build pipeline
├── build.ps1                 # Windows build pipeline
├── .github/workflows/
│   └── release.yml           # GitHub Actions release workflow
├── backend/
│   ├── cmd/owl/main.go       # Main binary (--port, --data-dir flags)
│   ├── internal/
│   │   ├── api/              # HTTP routes + handlers (under /api prefix)
│   │   ├── cluster/          # DBSCAN clustering
│   │   ├── config/           # Configuration
│   │   ├── db/               # Migration runner + SQL files
│   │   ├── embedding/        # LM Studio embedding client
│   │   ├── extractor/        # File content extraction
│   │   ├── intelligence/     # TF-IDF, strategies, suggester
│   │   ├── llm/              # LM Studio LLM client
│   │   ├── logging/          # Structured logging (tint + file)
│   │   ├── scanner/          # Filesystem walker
│   │   ├── store/            # SQLite CRUD queries
│   │   └── vector/           # Cosine similarity, vector math
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── api.ts            # REST client (auto-detects Tauri vs browser)
│   │   ├── app.tsx           # Root component + Refresh button
│   │   ├── components/       # FileTree, ProgressBar, Toast, FilePickerDialog
│   │   ├── hooks/            # TanStack Query hooks + toast system
│   │   ├── pages/            # Home, FileDetail, SuggestionDetail
│   │   ├── app.css
│   │   └── index.css
│   ├── src-tauri/            # Tauri v2 config + Rust source
│   │   ├── tauri.conf.json   # Sidecar config, capabilities, bundle settings
│   │   ├── src/lib.rs        # Sidecar spawn + lifecycle
│   │   ├── Cargo.toml
│   │   └── binaries/         # Go sidecar binary placed here during build
│   ├── vite.config.ts        # Vite config with /api proxy
│   └── package.json
├── data/                     # Runtime data (gitignored)
├── config.json               # User configuration
├── config.example.json
└── VISION.md                 # Design document
```

## TODO

- **Media support** — EXIF, audio/video metadata extraction
- **Content_tfidf only as default** — embeddings strategy requires LM Studio setup
