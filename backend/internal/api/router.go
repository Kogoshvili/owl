package api

import (
	"net/http"

	"owl/internal/api/handler"
	"owl/internal/api/middleware"
	"owl/internal/config"
	"owl/internal/extractor"
	"owl/internal/llm"
	"owl/internal/scanner"
	"owl/internal/store"
)

func NewRouter(s *store.Store, sc *scanner.Scanner, ext *extractor.Extractor, llmClient *llm.Client, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	wdh := handler.NewWatchedDirHandler(s, sc, ext)
	fh := handler.NewFileHandler(s, ext)
	ch := handler.NewCommentHandler(s)
	shh := handler.NewSuggestionHandler(s)
	ih := handler.NewIntelligenceHandler(s, llmClient, ext, cfg)

	mux.HandleFunc("GET /health", handleHealth)

	mux.HandleFunc("GET /watched-directories", wdh.List)
	mux.HandleFunc("POST /watched-directories", wdh.Create)
	mux.HandleFunc("PATCH /watched-directories/{id}", wdh.Update)
	mux.HandleFunc("DELETE /watched-directories/{id}", wdh.Delete)
	mux.HandleFunc("POST /watched-directories/{id}/scan", wdh.Scan)
	mux.HandleFunc("POST /watched-directories/{id}/extract", wdh.Extract)

	mux.HandleFunc("GET /files", fh.List)
	mux.HandleFunc("GET /files/extensions", fh.Extensions)
	mux.HandleFunc("GET /files/{id}", fh.Get)
	mux.HandleFunc("GET /files/{id}/raw", fh.Raw)
	mux.HandleFunc("GET /watched-directories/{id}/files", fh.ListByDir)
	mux.HandleFunc("POST /files/{id}/extract", fh.Extract)

	mux.HandleFunc("PUT /files/{id}/comment", ch.Upsert)
	mux.HandleFunc("DELETE /files/{id}/comment", ch.Delete)

	mux.HandleFunc("GET /suggestions", shh.List)
	mux.HandleFunc("POST /suggestions", shh.Create)
	mux.HandleFunc("GET /suggestions/{id}", shh.Get)
	mux.HandleFunc("PATCH /suggestions/{id}", shh.Update)
	mux.HandleFunc("DELETE /suggestions/{id}", shh.Delete)
	mux.HandleFunc("POST /suggestions/{id}/files", shh.AddFiles)
	mux.HandleFunc("DELETE /suggestions/{id}/files/{fileId}", shh.RemoveFile)
	mux.HandleFunc("POST /suggestions/{id}/materialize", shh.Materialize)

	mux.HandleFunc("GET /intelligence/strategies", ih.ListStrategies)
	mux.HandleFunc("GET /intelligence/folders/physical", ih.ListPhysicalFolders)
	mux.HandleFunc("GET /intelligence/folders/physical/files", ih.ListPhysicalFolderFiles)
	mux.HandleFunc("GET /intelligence/folders/physical/coherence", ih.AnalyzeFolderCoherence)
	mux.HandleFunc("GET /intelligence/folders/suggestions", ih.ListFolderSuggestions)
	mux.HandleFunc("POST /intelligence/folders/suggestions", ih.GenerateSuggestions)
	mux.HandleFunc("DELETE /intelligence/folders/suggestions/{id}", ih.DismissFolderSuggestion)
	mux.HandleFunc("POST /intelligence/folders/suggestions/refine-all", ih.RefineAllSuggestions)
	mux.HandleFunc("GET /intelligence/folders/suggestions/status", ih.GetGenerationStatus)
	mux.HandleFunc("POST /intelligence/refine/folder/{id}", ih.RefineFolder)
	mux.HandleFunc("GET /intelligence/files/unprocessed/count", ih.GetUnprocessedCount)
	mux.HandleFunc("GET /intelligence/files/processing-stats", ih.GetProcessingStats)
	mux.HandleFunc("GET /intelligence/folders/guards", ih.ListFolderGuards)
	mux.HandleFunc("PUT /intelligence/folders/guards", ih.SetFolderGuard)
	mux.HandleFunc("POST /intelligence/guard/run", ih.RunGuard)
	mux.HandleFunc("POST /intelligence/files/extract-orphans", ih.ExtractOrphans)

	return middleware.Logging(middleware.CORS(mux))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
