package api

import (
	"net/http"

	"owl/internal/api/handler"
	"owl/internal/api/middleware"
	"owl/internal/config"
	"owl/internal/extractor"
	"owl/internal/ollama"
	"owl/internal/scanner"
	"owl/internal/store"
)

func NewRouter(s *store.Store, sc *scanner.Scanner, ext *extractor.Extractor, cfg *config.Config, ollamaMgr *ollama.Manager) http.Handler {
	apiMux := http.NewServeMux()

	wdh := handler.NewWatchedDirHandler(s, sc, ext)
	fh := handler.NewFileHandler(s, ext)
	ch := handler.NewCommentHandler(s)
	shh := handler.NewSuggestionHandler(s)
	ih := handler.NewIntelligenceHandler(s, ext, cfg, ollamaMgr)

	apiMux.HandleFunc("GET /watched-directories", wdh.List)
	apiMux.HandleFunc("POST /watched-directories", wdh.Create)
	apiMux.HandleFunc("PATCH /watched-directories/{id}", wdh.Update)
	apiMux.HandleFunc("DELETE /watched-directories/{id}", wdh.Delete)
	apiMux.HandleFunc("POST /watched-directories/{id}/scan", wdh.Scan)
	apiMux.HandleFunc("POST /watched-directories/{id}/extract", wdh.Extract)

	apiMux.HandleFunc("GET /files", fh.List)
	apiMux.HandleFunc("GET /files/extensions", fh.Extensions)
	apiMux.HandleFunc("GET /files/{id}", fh.Get)
	apiMux.HandleFunc("GET /files/{id}/raw", fh.Raw)
	apiMux.HandleFunc("GET /watched-directories/{id}/files", fh.ListByDir)
	apiMux.HandleFunc("POST /files/{id}/extract", fh.Extract)

	apiMux.HandleFunc("PUT /files/{id}/comment", ch.Upsert)
	apiMux.HandleFunc("DELETE /files/{id}/comment", ch.Delete)

	apiMux.HandleFunc("GET /suggestions", shh.List)
	apiMux.HandleFunc("POST /suggestions", shh.Create)
	apiMux.HandleFunc("GET /suggestions/{id}", shh.Get)
	apiMux.HandleFunc("PATCH /suggestions/{id}", shh.Update)
	apiMux.HandleFunc("DELETE /suggestions/{id}", shh.Delete)
	apiMux.HandleFunc("POST /suggestions/{id}/files", shh.AddFiles)
	apiMux.HandleFunc("DELETE /suggestions/{id}/files/{fileId}", shh.RemoveFile)
	apiMux.HandleFunc("POST /suggestions/{id}/materialize", shh.Materialize)

	apiMux.HandleFunc("GET /intelligence/strategies", ih.ListStrategies)
	apiMux.HandleFunc("GET /intelligence/folders/physical", ih.ListPhysicalFolders)
	apiMux.HandleFunc("GET /intelligence/folders/physical/files", ih.ListPhysicalFolderFiles)
	apiMux.HandleFunc("GET /intelligence/folders/physical/coherence", ih.AnalyzeFolderCoherence)
	apiMux.HandleFunc("GET /intelligence/folders/suggestions", ih.ListFolderSuggestions)
	apiMux.HandleFunc("POST /intelligence/folders/suggestions", ih.GenerateSuggestions)
	apiMux.HandleFunc("DELETE /intelligence/folders/suggestions/{id}", ih.DismissFolderSuggestion)
	apiMux.HandleFunc("POST /intelligence/folders/suggestions/refine-all", ih.RefineAllSuggestions)
	apiMux.HandleFunc("GET /intelligence/folders/suggestions/status", ih.GetGenerationStatus)
	apiMux.HandleFunc("POST /intelligence/refine/folder/{id}", ih.RefineFolder)
	apiMux.HandleFunc("GET /intelligence/files/unprocessed/count", ih.GetUnprocessedCount)
	apiMux.HandleFunc("GET /intelligence/files/processing-stats", ih.GetProcessingStats)
	apiMux.HandleFunc("GET /intelligence/folders/guards", ih.ListFolderGuards)
	apiMux.HandleFunc("PUT /intelligence/folders/guards", ih.SetFolderGuard)
	apiMux.HandleFunc("POST /intelligence/guard/run", ih.RunGuard)
	apiMux.HandleFunc("GET /intelligence/guard/status", ih.GetGuardStatus)
	apiMux.HandleFunc("GET /intelligence/llm/status", ih.GetLlmStatus)
	apiMux.HandleFunc("POST /intelligence/llm/setup", ih.StartLlmSetup)
	apiMux.HandleFunc("GET /intelligence/llm/setup-status", ih.GetLlmSetupStatus)
	apiMux.HandleFunc("GET /intelligence/scan/status", wdh.GetScanStatus)
	apiMux.HandleFunc("GET /intelligence/extract/status", ih.GetExtractStatus)
	apiMux.HandleFunc("POST /intelligence/files/extract-orphans", ih.ExtractOrphans)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))

	return middleware.Logging(middleware.CORS(mux))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
