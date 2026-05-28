package handler

import (
	"encoding/json"
	"net/http"

	"owl/internal/store"
)

type VirtualFolderHandler struct {
	store *store.Store
}

func NewVirtualFolderHandler(s *store.Store) *VirtualFolderHandler {
	return &VirtualFolderHandler{store: s}
}

type createVirtualFolderRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateVirtualFolderRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type addFilesRequest struct {
	FileIDs []int64 `json:"file_ids"`
	Source  string  `json:"source"`
}

type materializeRequest struct {
	Path string `json:"path"`
}

func (h *VirtualFolderHandler) List(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	var sourcePtr *string
	if source != "" {
		sourcePtr = &source
	}
	folders, err := h.store.ListVirtualFolders(sourcePtr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if folders == nil {
		folders = []store.VirtualFolder{}
	}
	writeJSON(w, http.StatusOK, folders)
}

func (h *VirtualFolderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	detail, err := h.store.GetVirtualFolderDetail(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if detail == nil {
		writeError(w, http.StatusNotFound, "virtual folder not found")
		return
	}
	if detail.Files == nil {
		detail.Files = []store.File{}
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *VirtualFolderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createVirtualFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	folder, err := h.store.CreateVirtualFolder(req.Name, req.Description, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, folder)
}

func (h *VirtualFolderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req updateVirtualFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	folder, err := h.store.UpdateVirtualFolder(id, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if folder == nil {
		writeError(w, http.StatusNotFound, "virtual folder not found")
		return
	}
	writeJSON(w, http.StatusOK, folder)
}

func (h *VirtualFolderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteVirtualFolder(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *VirtualFolderHandler) AddFiles(w http.ResponseWriter, r *http.Request) {
	folderID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req addFilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	source := req.Source
	if source == "" {
		source = "manual"
	}

	if err := h.store.AddFilesToFolder(folderID, req.FileIDs, source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *VirtualFolderHandler) RemoveFile(w http.ResponseWriter, r *http.Request) {
	folderID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}
	fileID, ok := parsePathID(w, r, "fileId")
	if !ok {
		return
	}

	if err := h.store.RemoveFileFromFolder(folderID, fileID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *VirtualFolderHandler) Materialize(w http.ResponseWriter, r *http.Request) {
	folderID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req materializeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	if err := h.store.SetMaterialized(folderID, req.Path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	folder, _ := h.store.GetVirtualFolder(folderID)
	writeJSON(w, http.StatusOK, folder)
}
