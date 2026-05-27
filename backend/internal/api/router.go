package api

import (
	"net/http"

	"owl/internal/api/handler"
	"owl/internal/api/middleware"
	"owl/internal/scanner"
	"owl/internal/store"
)

func NewRouter(s *store.Store, sc *scanner.Scanner) http.Handler {
	mux := http.NewServeMux()

	wdh := handler.NewWatchedDirHandler(s, sc)
	fh := handler.NewFileHandler(s)
	ch := handler.NewCommentHandler(s)
	th := handler.NewTagHandler(s)
	vfh := handler.NewVirtualFolderHandler(s)
	nh := handler.NewNoteHandler(s)
	sh := handler.NewSearchHandler(s)

	mux.HandleFunc("GET /health", handleHealth)

	mux.HandleFunc("GET /watched-directories", wdh.List)
	mux.HandleFunc("POST /watched-directories", wdh.Create)
	mux.HandleFunc("PATCH /watched-directories/{id}", wdh.Update)
	mux.HandleFunc("DELETE /watched-directories/{id}", wdh.Delete)
	mux.HandleFunc("POST /watched-directories/{id}/scan", wdh.Scan)

	mux.HandleFunc("GET /files", fh.List)
	mux.HandleFunc("GET /files/{id}", fh.Get)
	mux.HandleFunc("GET /watched-directories/{id}/files", fh.ListByDir)

	mux.HandleFunc("PUT /files/{id}/comment", ch.Upsert)
	mux.HandleFunc("DELETE /files/{id}/comment", ch.Delete)

	mux.HandleFunc("GET /tags", th.List)
	mux.HandleFunc("POST /files/{id}/tags", th.AddFileTag)
	mux.HandleFunc("DELETE /files/{id}/tags/{tagId}", th.RemoveFileTag)
	mux.HandleFunc("POST /notes/{id}/tags", th.AddNoteTag)
	mux.HandleFunc("DELETE /notes/{id}/tags/{tagId}", th.RemoveNoteTag)

	mux.HandleFunc("GET /virtual-folders", vfh.List)
	mux.HandleFunc("POST /virtual-folders", vfh.Create)
	mux.HandleFunc("GET /virtual-folders/{id}", vfh.Get)
	mux.HandleFunc("PATCH /virtual-folders/{id}", vfh.Update)
	mux.HandleFunc("DELETE /virtual-folders/{id}", vfh.Delete)
	mux.HandleFunc("POST /virtual-folders/{id}/files", vfh.AddFiles)
	mux.HandleFunc("DELETE /virtual-folders/{id}/files/{fileId}", vfh.RemoveFile)
	mux.HandleFunc("POST /virtual-folders/{id}/materialize", vfh.Materialize)

	mux.HandleFunc("GET /notes", nh.List)
	mux.HandleFunc("POST /notes", nh.Create)
	mux.HandleFunc("GET /notes/{id}", nh.Get)
	mux.HandleFunc("PATCH /notes/{id}", nh.Update)
	mux.HandleFunc("DELETE /notes/{id}", nh.Delete)
	mux.HandleFunc("POST /notes/{id}/materialize", nh.Materialize)

	mux.HandleFunc("GET /search", sh.Search)

	return middleware.Logging(middleware.CORS(mux))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
