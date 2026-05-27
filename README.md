# Owl File Manager

A desktop application for automatic file and note management. Instead of manually organizing files and notes, the system analyzes your content and creates an intelligent interface to access everything structurally.

## Core Problem

- Files get scattered across Downloads, Desktop, Documents, etc. and never get organized.
- Notes apps require manual maintenance — you have to tag, folder, and sort everything yourself.
- There's no easy way to connect related notes and files together.

## Design Direction: Notes as File Context

Owl's identity is a **smart file manager** — not a notes app competing with Obsidian. Notes exist to annotate and enrich your files, not as their own universe.

**v1:** Notes are a feature of the file manager. Lightweight markdown editor. Notes attach to virtual folders and can be materialized into `.md` files on disk. Comments provide quick per-file annotations. You can browse all notes standalone, but the main entry is always through the file context.

**Future:** If notes usage grows, notes can split into their own workspace with dedicated features (backlinks, graph view, rich editor) while still sharing the same data model and Projects system. This lets Owl become "Obsidian with files" — a notes app that natively handles PDFs, images, docs alongside markdown.

## Use Cases

### Smart File Organization
You have PDFs, docs, images, etc. scattered across folders from past projects. Owl will:
- Analyze your specified folders (Downloads, Documents, etc.)
- Create searchable indexes based on file names and content
- Auto-generate tags and categories (e.g., "all PDFs about birds")
- Run in the background and watch selected folders for changes
- Create **virtual folders** — groupings that only exist in the Owl UI, collecting related files from different locations into one place

**Example:** You've been researching birds in your area. You have some files on your Desktop, others in Downloads. Owl creates a virtual folder called "Bird Research" that shows everything in one view.

### Intelligent Notes & Comments
**Notes** are markdown documents attached to virtual folders — if you took notes about birds, they'll show up alongside your files when you open your Bird Research virtual folder. Notes can be **materialized** into actual `.md` files on disk.

**Comments** are lightweight per-file annotations — quick context or reminders attached to individual files.

## Core Concepts

- **Virtual Folders** — Collections of related files from across your system, visible only in the Owl interface. No files are moved. If a user finds a virtual folder useful, they can **materialize** it — a real folder is created on disk and all the files are moved there. Auto-suggested folders appear alongside manually created ones for easy review.
- **Tags** — Labels that mark files and notes by context or type. Auto-generated from content analysis, or manually created. Auto tags are reviewed before accepting.
- **Comments** — Lightweight per-file annotations for quick context. One comment per file.
- **Notes** — Markdown documents attached to virtual folders. Can be materialized into actual `.md` files on disk.
- **Projects** *(v2)* — Workspaces combining multiple virtual folders and notes.

## Stack

- **Backend:** Go + SQLite (FTS5)
- **Frontend:** Tauri v2 + Preact + TypeScript
