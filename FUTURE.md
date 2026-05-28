
## v2 - Future

### Dashboard
- [ ] Proper dashboard page (stats, recent activity, quick actions)

### Automation
- [ ] Batch operations - select multiple files, bulk tag/extract/delete
- [ ] Auto-scan on startup
- [ ] Directory watching (fsnotify) for real-time file changes
- [ ] Background pipeline coordination — scan → extract → tag → suggest

### Intelligence
- [ ] User feedback loop — confirm/dismiss suggestions, learn from feedback
- [ ] Min file count thresholds configurable via settings (currently hardcoded: tags=3, folders=3)
- [ ] `add_to_folder` suggestion type: suggest adding orphan files to an existing coherent virtual folder
- [ ] `merge_folders` suggestion type: suggest merging two similar sibling folders
- [ ] UI drag-and-drop: drag file into folder (add), drag folder into folder (merge)

### Notes & Materialization
- [ ] Notes frontend (backend done)
  - Create/edit/delete notes
  - Attach notes to virtual folders
  - Tag notes
- [ ] Materialization — create folder on disk + move/copy files (backend stub exists)
- [ ] Note materialization — write .md files to disk (backend stub exists)

### Media & Desktop
- [ ] Image metadata extraction outside extraction pipeline
- [ ] EXIF data extraction for JPEG
- [ ] Audio/video metadata extraction
- [ ] Thumbnail generation for images
- [ ] OCR + AI vision for image content understanding
- [ ] Tauri desktop integration - system tray, native file dialogs
- [ ] Projects feature

### Tags (v1.2)
- [ ] Redesign tags system: remove auto-tagging, move to manual only, allow attaching notes to tags
