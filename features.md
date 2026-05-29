# Owl File Manager — Features

## V1 (Current)

### Directory Management
- Add/remove watched directories (e.g., `~/Downloads`, `~/Documents`)
- Manual scan: records every file's path, name, extension, size, modification time
- Scan per directory or globally

### Content Extraction
- Manual trigger per directory or per file
- Supported formats: txt, md, pdf, docx, xlsx, pptx, csv, json, xml, yaml, html, css, scss, svg, code files (go, py, js, ts, rs, java, c, cpp, rb, php, sql, sh, bat, ps1) and more
- Binary/unsupported files fall back to filename-based keyword extraction
- Extracted text stored in FTS5 virtual tables for full-text search

### Search
- Unified search across 4 scopes: filenames, content, comments, tags
- Scope toggle pills to narrow results
- Ranked results with snippets and match-source badges
- Click results to navigate directly to file detail

### Physical Folder Tree
- Real filesystem hierarchy displayed in the UI
- Expand/collapse with depth-based auto-expand
- File counts per folder
- Top-level files shown inline when expanding nodes
- Coherence badges (coherent/incoherent) on folders

### LLM Folder Guard
- Guards classify folders as "related" (files belong together — skip) or "open" (files are fair game — process)
- BFS top-down traversal, LLM classifies each folder
- Max depth of 3 — deeper folders auto-guarded
- Parent context passed to LLM for better decisions
- User can toggle guard status per folder (overrides LLM)
- When a parent is guarded, stale descendant guards are cleaned up

### Coherence Analysis
- Per-folder TF-IDF vector computation
- Average pairwise cosine similarity as coherence score
- Threshold-based classification (coherent vs incoherent)
- Outlier detection within coherent folders
- Orphans collected from: root files, incoherent folder files, outliers

### Virtual Folder Suggestions
- Generation pipeline:
  1. Build global TF-IDF corpus (content + filename fallback)
  2. Run folder guard classification
  3. Filter guarded folders
  4. Analyze coherence on remaining folders
  5. Collect orphans
  6. Run strategy on orphans
  7. Cluster, name suggestions, save to DB
- Two strategies (configurable):
  - **content_tfidf** (default): TF-IDF vectors + pairwise cosine similarity + flood-fill clustering. ~30s for 12K files.
  - **embeddings**: LM Studio embedding vectors + DBSCAN clustering. ~20-40min for 12K first run, cached thereafter.
- Suggestion naming uses top TF-IDF terms (LLM refinement on user click)
- Generation runs async — status tracked in memory, pollable via API
- Progress stages: initializing → classifying_folders → building_corpus → analyzing_coherence → clustering → saving → complete

### Suggestion Lifecycle
- **Accept** — promotes from `auto` to `manual` source
- **Dismiss** — deletes the suggestion
- **Refine** — LLM renames and adds description (per folder or bulk refine all)
- Suggestions persist in the database across restarts

### Tags
- Auto-generated from content analysis
- Accept/dismiss flow (same as suggestions)
- LLM refinement evaluates for specificity, renames vague tags, adds descriptions
- Tags have `source` field: `auto` or `manual`
- Tag files manually via file detail page

### Comments
- One lightweight text annotation per file
- Create/edit/delete via file detail page
- Included in search scope

### Folder Guard UI
- Folder tree shows guarded (🔒) and open (🔓) badges
- Click badge to toggle guard status
- User toggles saved as `source='user'`, always respected

### Configuration
- JSON config file at project root (`config.json`) or `~/.config/owl/config.json`
- Environment variable overrides: `LLM_ENABLED`, `LLM_BASE_URL`, `LLM_MODEL`, `EMBED_MODEL`, `LOG_LEVEL`
- Config options: LLM settings, embedding model, folder strategy selection

### API
- REST API on port 3721 (stdlib `http.ServeMux`)
- 30+ endpoints covering files, directories, tags, virtual folders, search, intelligence
- JSON responses throughout

---

## V2 (Planned)

### Dashboard
- Proper dashboard page with stats, recent activity, quick actions

### Automation
- Real-time file watching via fsnotify — no manual scan needed
- Auto-scan on startup
- Background pipeline: scan → extract → tag → suggest
- Batch operations: select multiple files, bulk tag/extract/delete

### Media & Desktop
- Image metadata extraction (EXIF, dimensions)
- Audio/video metadata extraction
- Thumbnail generation for images
- OCR + AI vision for image content understanding
- Tauri system tray integration, native file dialogs

### Projects
- Workspaces combining multiple virtual folders and notes
- Project-level tags and search scoping

### Smart Suggestions
- `add_to_folder` type: suggest adding orphan files to existing coherent virtual folders
- `merge_folders` type: suggest merging two similar sibling folders
- UI drag-and-drop: drag file into folder (add), drag folder into folder (merge)

### Tags (v1.2)
- Redesign tags system: remove auto-tagging, move to manual only
- Allow attaching notes to tags

### Notes
- Create/edit/delete notes (backend stub existed, removed in v1.1 cleanup)
- Attach notes to virtual folders
- Tag notes
- Materialization: write notes as .md files on disk
