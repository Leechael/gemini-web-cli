package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

type notebookCreateRequest struct {
	Title string `json:"title"`
}

type notebookSourceRequest struct {
	Path string `json:"path,omitempty"`
	URL  string `json:"url,omitempty"`
}

type notebookSourceJSON struct {
	Resource   string `json:"resource"`
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	UploadedAt int64  `json:"uploaded_at_unix,omitempty"`
}

type notebookJSON struct {
	Resource    string               `json:"resource"`
	Title       string               `json:"title"`
	Emoji       string               `json:"emoji,omitempty"`
	Sources     []notebookSourceJSON `json:"sources"`
	SourceCount int                  `json:"source_count"`
	CreatedUnix int64                `json:"created_unix,omitempty"`
	UpdatedUnix int64                `json:"updated_unix,omitempty"`
}

func notebookToJSON(nb *rpcs.Notebook) notebookJSON {
	sources := make([]notebookSourceJSON, 0, len(nb.Sources))
	for _, src := range nb.Sources {
		sources = append(sources, notebookSourceJSON{
			Resource:   src.ResourceName,
			FileName:   src.FileName,
			MimeType:   src.MimeType,
			UploadedAt: src.UploadedUnix,
		})
	}
	return notebookJSON{
		Resource:    nb.ResourceName,
		Title:       nb.Title,
		Emoji:       nb.Emoji,
		Sources:     sources,
		SourceCount: len(sources),
		CreatedUnix: nb.CreatedUnix,
		UpdatedUnix: nb.UpdatedUnix,
	}
}

// handleNotebookCreate handles POST /v1/notebooks.
func (s *Server) handleNotebookCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req notebookCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	resource, err := s.client.CreateNotebook(r.Context(), req.Title)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if resource == "" {
		writeError(w, http.StatusBadGateway, "create notebook returned empty resource")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resource": resource, "title": req.Title})
}

// handleNotebookGet handles GET /v1/notebooks/{id}.
func (s *Server) handleNotebookGet(w http.ResponseWriter, r *http.Request) {
	nb, err := s.client.GetNotebook(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if nb == nil {
		writeError(w, http.StatusNotFound, "notebook not found")
		return
	}
	writeJSON(w, http.StatusOK, notebookToJSON(nb))
}

// handleNotebookChats handles GET /v1/notebooks/{id}/chats.
func (s *Server) handleNotebookChats(w http.ResponseWriter, r *http.Request) {
	items, err := s.client.ListNotebookChats(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	type chatItem struct {
		Cid     string `json:"cid"`
		Title   string `json:"title"`
		Updated string `json:"updated,omitempty"`
	}
	out := make([]chatItem, 0, len(items))
	for _, it := range items {
		out = append(out, chatItem{Cid: it.Cid, Title: it.Title, Updated: it.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{"chats": out})
}

// handleNotebookAddSource handles POST /v1/notebooks/{id}/sources.
func (s *Server) handleNotebookAddSource(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req notebookSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	id := r.PathValue("id")
	ctx := r.Context()

	var nb *rpcs.Notebook
	var err error
	switch {
	case req.URL != "":
		if !client.IsHTTPURL(req.URL) {
			writeError(w, http.StatusBadRequest, "url must be an absolute http or https URL with a host")
			return
		}
		nb, err = s.client.AddNotebookURLSource(ctx, id, req.URL)
	case req.Path != "":
		up, upErr := s.client.UploadFile(ctx, req.Path)
		if upErr != nil {
			writeError(w, http.StatusBadGateway, "upload failed: "+upErr.Error())
			return
		}
		nb, err = s.client.AddNotebookSource(ctx, id, up.FileName, up.MimeType, up.ID)
	default:
		writeError(w, http.StatusBadRequest, "either path or url is required")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if nb == nil {
		writeError(w, http.StatusBadGateway, "empty notebook response")
		return
	}
	writeJSON(w, http.StatusOK, notebookToJSON(nb))
}

// handleNotebookRemoveSource handles DELETE /v1/notebooks/{id}/sources/{sid}.
func (s *Server) handleNotebookRemoveSource(w http.ResponseWriter, r *http.Request) {
	resource := "notebooks/" + r.PathValue("id") + "/sources/" + r.PathValue("sid")
	if err := s.client.RemoveNotebookSource(r.Context(), resource); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"removed": resource})
}
