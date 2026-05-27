# Owl File Manager

A desktop application for automatic file and note management. Instead of manually organizing files and notes, the system analyzes your content and creates an intelligent interface to access everything structurally.

## Core Problem

- Files get scattered across Downloads, Desktop, Documents, etc. and never get organized.
- Notes apps require manual maintenance — you have to tag, folder, and sort everything yourself.
- There's no easy way to connect related notes and files together.

## Design Direction: Notes as File Context

Owl's identity is a **smart file manager** — not a notes app competing with Obsidian. Notes exist to annotate and enrich your files, not as their own universe.

**v1 (Model C):** Notes are a feature of the file manager. Lightweight markdown editor. Notes attach to virtual folders and can be materialized into `.md` files on disk. Comments provide quick per-file annotations. You can browse all notes standalone, but the main entry is always through the file context.

**Future (Model B):** If notes usage grows, notes can split into their own workspace with dedicated features (backlinks, graph view, rich editor) while still sharing the same data model and Projects system. This lets Owl become "Obsidian with files" — a notes app that natively handles PDFs, images, docs alongside markdown.

## Use Cases

### 1. Smart File Organization
You have PDFs, docs, images, etc. scattered across folders from past projects. Owl will:
- Analyze your specified folders (Downloads, Documents, etc.)
- Create searchable indexes based on file names and content
- Auto-generate tags and categories (e.g., "all PDFs about birds")
- Run in the background and watch selected folders for changes. When a new file is added to a watched folder (e.g., Desktop), Owl can suggest adding it to an existing virtual folder or update tags if file content changes.
- Create **virtual folders** — groupings that only exist in the Owl UI, collecting related files from different locations into one place

**Example:** You've been researching birds in your area. You have some files on your Desktop, others in Downloads. Owl creates a virtual folder called "Bird Research" that shows everything in one view.

### 2. Intelligent Notes & Comments
**Notes** are markdown documents attached to virtual folders — if you took notes about birds, they'll show up alongside your files when you open your Bird Research virtual folder. Notes can be **materialized** into actual `.md` files on disk.

**Comments** are lightweight per-file annotations — quick context or reminders attached to individual files.

## Core Concepts

- **Virtual Folders** — Collections of related files from across your system, visible only in the Owl interface. No files are actually moved. If a user finds a virtual folder useful, they can **materialize** it — a real folder is created on disk and all the files are moved there.
- **Tags** — Auto-generated labels that mark files and notes by context or type.
- **Comments** — Lightweight per-file annotations for quick context. One comment per file.
- **Notes** — Markdown documents attached to virtual folders. Can be materialized into actual `.md` files on disk.
- **Projects** *(v2)* — Workspaces combining multiple virtual folders and notes.

## Implementation

### Stack
- **Backend:** Go — handles file watching, indexing, content analysis, tagging, and all background processing.
- **Frontend:** Tauri + Preact — native desktop UI that communicates with the Go backend.

### Database & Storage
- **SQLite** as the main database for metadata, tags, file paths, virtual folder structure, etc.
- **Full-text search (FTS)** via SQLite FTS5 extension for searching file names and text content.

### Determining File Relationships (Virtual Folders)

**Recommended approach for v1: SQLite FTS5 + keyword overlap scoring.**

Start simple. A vector DB or LLM is overkill for a desktop app with a few thousand files, and they risk false connections that break user trust. FTS is transparent — the user sees *why* files are grouped (shared keywords).

How it works:
- Index file content with SQLite FTS5
- Score relationships by counting shared significant terms between files
- Add simple heuristics: same source folder + similar file names = likely related
- Let users confirm or dismiss suggestions and learn from that feedback

If FTS hits a limit later, add a local vector DB (e.g., `chromem-go`, no server needed) for semantic matching. LLM-based suggestions can be an optional on-demand feature, not part of background scanning.

### File Content Indexing
- Text files, PDFs, docs — extract text and index with FTS5.
- Images — OCR + AI vision to understand content for better categorization and tagging.


