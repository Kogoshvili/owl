# Owl File Manager — Application Flow

## Overview

```
Add directories → Scan → Extract → Browse → Generate suggestions → Review → Search
```

---

## 1. Add Watched Directories

User adds directories via the Ingest page (e.g., `C:\Users\...\Downloads\`). Each directory becomes a watched directory record in the database. There is no real-time filesystem watching — all operations are manual triggers.

---

## 2. Scan

User clicks **Scan** on a directory (or uses the global scan endpoint). The scanner walks the directory recursively and records every file into the `files` table:

| Field | Source |
|-------|--------|
| `path` | Full absolute path |
| `name` | Filename |
| `extension` | File extension |
| `size` | File size in bytes |
| `parent_dir` | Directory containing the file |
| `watched_dir_id` | FK to watched directory |
| `status` | `'active'` (or `'deleted'` if removed during rescan) |
| `modified_at` | Filesystem modification timestamp |
| `processing_status` | `'unprocessed'` initially |

Files that no longer exist on disk are marked `status = 'deleted'` during rescan (soft delete).

---

## 3. Extract

User clicks **Extract** on a directory (or per-file). The extractor processes files with supported extensions:

1. Reads file content (up to 50MB, extracts up to 500KB of text)
2. For supported formats (txt, md, pdf, docx, xlsx, code files, etc.): parses and extracts readable text
3. For unsupported formats (exe, zip, jpg, png, mp4, etc.): no content extracted, `processing_status` set to `'failed'` or left as `'unprocessed'`
4. Stores extracted text in `files_fts` FTS5 virtual table
5. Sets `processing_status` to `'processed'` on success, `'failed'` on error

Files without extracted content (unsupported or failed) fall back to **filename-based keyword extraction** during analysis — tokens extracted from the filename (split on separators, filtered for stopwords/short words/numbers) plus an extension-derived content tag (e.g., "executable", "image", "document").

---

## 4. Browse

The **Physical Folder Tree** shows the real filesystem hierarchy built from all distinct `parent_dir` values in the database. Each node shows:

- Folder name and full path
- File count
- Guard status (🔒 guarded / 🔓 open)
- Coherence badge (if analyzed)
- Files within the folder (fetched on expand)

Users can toggle guard status directly on tree nodes.

---

## 5. Generate Suggestions

This is the core intelligence pipeline, triggered by clicking **Generate** on the Suggestions page. It runs asynchronously — the UI receives a 202 response and the backend processes in a goroutine.

### Stage 1: Initialize
- Clear old auto-generated suggestions from the database
- Load the physical folder tree for all watched directories

### Stage 2: Classify Folders (LLM Guard)
- BFS traversal of the physical folder tree, starting from depth 1
- For each folder (up to `maxGuardDepth = 3`):
  - Already classified? Use existing decision. Guarded → skip subtree.
  - Guarded by ancestor? Skip subtree.
  - Deeper than max depth? Auto-guard (mark as guarded).
  - Otherwise: call LLM with folder name, file names, subfolder names, parent context
  - LLM responds: `{"related": true/false, "reason": "..."}`
  - `related: true` → guarded (files belong together, skip subtree)
  - `related: false` → open (files are random, process children)
- If parent is newly guarded, clean stale descendant classifications from DB

### Stage 3: Build Global Corpus
- Collect all file IDs from unguarded folders
- Build a TF-IDF corpus across all these files:
  - Content keywords (from `files_fts` table) if available
  - Otherwise: filename tokens + extension tag (fallback)
- Compute term frequencies, document frequencies, and TF-IDF scores
- Store as a `Corpus` struct with keyword maps per file ID

### Stage 4: Analyze Coherence
- For each unguarded folder with ≥3 files:
  - Compute average pairwise cosine similarity of TF-IDF vectors
  - If similarity ≥ threshold: folder is **coherent** (files belong together)
  - Identify outlier files within coherent folders
  - If similarity < threshold: folder is **incoherent**, all files become candidates

### Stage 5: Collect Orphans
Files that need organizational help:
1. Files from incoherent folders (all files)
2. Outlier files from coherent folders
3. Files at the watched directory root (not in any subfolder)

### Stage 6: Run Strategy

**Content TF-IDF strategy** (default):
1. Build TF-IDF corpus specifically for orphan files
2. Compute pairwise cosine similarity for all orphan pairs
3. Keep pairs with similarity ≥ 0.45
4. Flood-fill clustering: start with a file, add all similar files, repeat transitively
5. Clusters ≥ 3 files become suggestions
6. Sub-cluster large clusters with a boosted threshold
7. Name each cluster using top 3 TF-IDF terms ("Setup files", "Financial reports")

**Embeddings strategy** (if configured):
1. For each orphan file, build embedding text: `name + extension + parent_dir + content_preview`
2. Compute vector embeddings via LM Studio `/v1/embeddings` (batches of 20)
3. Cache embeddings in `file_embeddings` table (BLOB) for reuse
4. Run DBSCAN clustering (eps=0.4, minPts=3) on the embedding vectors
5. Name using TF-IDF top terms from the global corpus

### Stage 7: Save Suggestions
- Each cluster is saved as a `virtual_folder` with `source='auto'`, `suggestion_type='new_folder'`
- File-to-folder mappings saved in `virtual_folder_files`
- Suggestions appear in the UI after refresh

---

## 6. Review Suggestions

Each suggestion card shows:
- Auto-generated name and description
- Confidence score (0-1)
- File count and preview list
- Action buttons

Actions:
- **Refine** — calls LLM to evaluate the cluster, rename, add description, and optionally remove outlier files. Only happens on user click, not during generation.
- **Accept** — updates `source` from `'auto'` to `'manual'`, making it a permanent virtual folder
- **Dismiss** — deletes the virtual folder and its file mappings
- **Refine All** — runs LLM refinement on all auto-suggestions in one batch

---

## 7. Search

The search endpoint queries across 4 configurable scopes:
- **Filenames** — `LIKE` query on `files.name` and `files.path`
- **Content** — FTS5 full-text search on `files_fts` with ranking and snippets
- **Comments** — `LIKE` query on `comments.content`
- **Tags** — `LIKE` query on `tags.name`, joined through `file_tags`

Results are deduplicated and merged. Each result shows matched scopes as color-coded badges.

---

## Configuration Flow

```
config.json (project root / ~/.config/owl/)
    ↓
owl/internal/config/config.go — loads JSON, applies env var overrides
    ↓
Used by: server setup, handler construction, strategy registry
```

Key settings:
- `llm.base_url` — LM Studio server URL
- `llm.model` — LLM model for guard + refinement
- `llm.embed_model` — Embedding model for embeddings strategy
- `llm.folder_strategy` — `"content_tfidf"` or `"embeddings"`

---

## Startup Sequence

1. Load config (JSON + env vars)
2. Open SQLite database at `data/owl.db`
3. Run pending migrations (golang-migrate with embed.FS)
4. Initialize store, scanner, extractor, LLM client, embedding client
5. Register strategies (content_tfidf always, embeddings if configured)
6. Set up HTTP routes on `:3721`
7. Start HTTP server
