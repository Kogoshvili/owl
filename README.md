# Owl File Manager

A desktop application for automatic file organization. Instead of manually sorting files across Downloads, Desktop, and Documents, Owl analyzes your content and creates an intelligent interface to access everything structurally.

## Core Problem

- Files get scattered across folders and never get organized.
- Manual organization is tedious and never keeps up.
- There's no easy way to find related files across different directories.

## Design Direction

Owl is a **smart file manager** — not a search tool or a notes app. It understands your files' content, recognizes when files are related, and surfaces intelligent suggestions for grouping them. You stay in control: suggest, accept, dismiss, refine.

**v1 (current):** Smart file organization. Add watched directories, scan files, extract content, browse the real folder hierarchy, and generate intelligent grouping suggestions. Accept suggestions to create virtual folders, dismiss what doesn't fit, refine names with LLM help.

**v2 (planned):** Automation, media support, and richer organization tools. Real-time file watching, batch operations, image/audio/video metadata, drag-and-drop organization, materialization of virtual folders to disk, and project workspaces.

## Use Cases

### Smart File Organization
You have PDFs, documents, images, and installers scattered across Downloads and Desktop. Owl will:
- Scan your watched directories and index every file
- Extract text content from supported formats (PDF, DOCX, code, etc.)
- Classify folders as coherent (well-organized — leave alone) or needing attention
- Run analysis strategies on orphan files and suggest virtual folder groupings
- Let you accept, dismiss, or refine suggestions with LLM-assisted naming

**Example:** You've downloaded research papers, meeting notes, and installer files over months. Owl identifies groups like "Q4 Financial Reports," "Python Project Assets," and "Meeting Notes 2024" — without moving a single file on disk.

### Intelligent Discovery
- Browse your real filesystem tree directly in the UI, with file counts per folder
- Search across filenames, extracted content, comments, and tags in one query
- See coherence indicators on folders (are the files inside related or scattered?)
- Review auto-generated suggestions and decide what to keep

## Core Concepts

- **Watched Directories** — Folders you tell Owl to monitor. Scanned manually, files recorded in the database.
- **Physical Folder Tree** — The real filesystem hierarchy, displayed with file counts and coherence indicators.
- **Content Extraction** — Text is extracted from supported file types and stored for search and analysis.
- **Virtual Folders** — Collections of related files visible only in Owl. Created manually or auto-suggested. No files are moved on disk.
- **Folder Guard** — LLM classifies folders as "related" (app/project files, skip) or "open" (random files, process). Prevents breaking up well-organized folders.
- **Coherence Analysis** — Measures how similar files are within a folder. Coherent folders are left alone. Incoherent folders contribute files to the suggestion pool.
- **Tags** — Labels marking files by context. Auto-generated from content, reviewed before accepting.
- **Comments** — Lightweight per-file annotations for quick context.
- **Strategies** — Two organization backends: `content_tfidf` (fast, TF-IDF + cosine similarity) and `embeddings` (semantic, LM Studio + DBSCAN).

## Stack

- **Backend:** Go + SQLite (FTS5) + stdlib HTTP
- **Frontend:** Tauri v2 + Preact + TypeScript
- **LLM:** LM Studio (OpenAI-compatible) for folder guard + refinement
- **Embeddings:** LM Studio `/v1/embeddings` for semantic strategy
