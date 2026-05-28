# Owl File Manager — Features & Functionality

## Overview

Owl is a desktop file manager that helps users organize files on their local filesystem. It watches directories the user selects, indexes file content and metadata, and provides intelligent suggestions for organizing files into virtual folders. The backend is Go with SQLite, the frontend is Preact + TypeScript, and the desktop shell is Tauri v2.

---

## Core Flow

### 1. Watch & Scan
User adds directories to watch (e.g., `~/Downloads`, `~/Documents`). Owl scans them recursively, recording every file's path, name, extension, size, and modification time into SQLite. Scans are triggered manually via the Ingest page — Owl does not watch for filesystem changes in real-time.

### 2. Extract & Index
After scanning, content extraction runs (also manual). Owl reads file content for supported formats (txt, md, pdf, docx, xlsx, pptx, json, xml, yaml, code files, etc.) and stores the text in an FTS5 virtual table for full-text search. File metadata (dimensions for images, page count for PDFs, etc.) is stored as JSON blobs.

### 3. Search
Unified search across filenames, extracted content, comments, tags, and notes. User types a query, selects which scopes to search via toggle pills, and gets ranked results with snippets.

### 4. Intelligent Organization
This is Owl's core value. The system analyzes files and suggests how to organize them:

#### Tags
Auto-generated labels for files based on content analysis. Tags have a source (`auto` or `manual`). Auto tags can be accepted (promoted to manual) or dismissed (deleted). LLM refinement evaluates auto tags for meaningfulness, renames vague ones, and adds descriptions.

**Status:** Needs redesign — currently produces too many generic tags. Deferred to v1.2.

#### Virtual Folders
Collections of files that don't need to be in the same physical directory. Virtual folders can be created manually or suggested automatically. Auto-suggested folders go through the same accept/dismiss/refine flow as tags. Folders can be "materialized" — written to disk as real directories.

---

## Folder Intelligence (v1.1 — Current Focus)

### Problem
A user watches `~/Downloads`. Inside there are:
- Subfolders that are already well-organized (e.g., `animals/dogs/`, `animals/cats/`, `project-x/`) — these came from unzipped archives or intentional organization
- Orphan files scattered in the root (e.g., `report_q1.pdf`, `report_q2.pdf`, `data.csv`) — these need grouping
- Some subfolders that are incoherent (user dumped random files into a "stuff" folder)

The old system clustered all files blindly, breaking apart already-related subfolder contents and creating useless suggestions.

### New Approach

#### Physical Folder Tree
Owl discovers and displays the real filesystem folder hierarchy. For each watched directory, it queries all distinct `parent_dir` values from the files table and builds a tree. The UI shows this tree with expand/collapse, file counts, and coherence indicators.

#### Coherence Analysis
Each physical subfolder is analyzed for content coherence:
- Compute TF-IDF vectors for all files in the folder
- Calculate average pairwise cosine similarity
- Folders with high similarity → coherent (shown green, left alone)
- Folders with low similarity → incoherent (shown yellow/red, files become candidates)

#### Orphan Collection
Files that need organizational help:
1. **Root orphans** — files directly in the watched directory root (not in any subfolder)
2. **Incoherent subfolder files** — files in subfolders that failed coherence check
3. **Outlier files** — individual files in otherwise-coherent folders that don't match the theme

#### Typed Suggestions
Instead of only "create new folder", the system generates three types:

1. **`new_folder`** — A cluster of orphan files that are semantically related. "Create a 'Statistics USA' folder with these 8 PDFs."
2. **`add_to_folder`** — An orphan file that matches an existing physical or virtual folder. "Add `report_q3.pdf` to the 'Statistics USA' folder."
3. **`merge_folders`** — Two or more adjacent subfolders with very similar content. "Merge `dogs/` and `cats/` into a 'pets' folder."

Each suggestion has a confidence score and can be accepted or dismissed.

---

## Strategies

Two folder suggestion strategies, selectable via config:

### Content TF-IDF (default, fast)
- Runs TF-IDF on extracted file content from the FTS table
- Computes pairwise cosine similarity between TF-IDF vectors
- Clusters using flood-fill on similarity edges
- ~30 seconds for 12K files
- Good for quick iterations

### Embeddings (better quality, slower)
- Generates vector embeddings via LM Studio's OpenAI-compatible API (`/v1/embeddings`)
- Uses file name + extension + parent dir + first 2000 chars of content as input
- Clusters using DBSCAN (pure Go implementation)
- Caches embeddings in SQLite `file_embeddings` table (BLOB) for instant re-runs
- ~20-40 minutes for 12K files (first run), instant on subsequent runs
- Better at semantic similarity ("quarterly report" ≈ "financial summary")

---

## Configuration

```json
{
  "llm": {
    "enabled": true,
    "base_url": "http://localhost:1234/v1",
    "model": "model-name",
    "embed_model": "text-embedding-model",
    "folder_strategy": "content_tfidf"
  }
}
```

Environment variable overrides: `LLM_ENABLED`, `LLM_BASE_URL`, `LLM_MODEL`, `EMBED_MODEL`

---

## Technology Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.26, stdlib `http.ServeMux` |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| Migrations | `golang-migrate/v2` with `embed.FS` |
| Frontend | Preact + Vite + TypeScript |
| Desktop | Tauri v2 |
| LLM | LM Studio (OpenAI-compatible API) |
| Embeddings | LM Studio `/v1/embeddings` endpoint |
| Clustering | DBSCAN (pure Go) |
| Logging | `log/slog` + `tint` handler |
