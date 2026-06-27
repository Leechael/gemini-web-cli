package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

type researchRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model,omitempty"`
}

type researchCreateResponse struct {
	ID      string   `json:"id"`
	ChatID  string   `json:"chat_id"`
	Title   string   `json:"title,omitempty"`
	ETAText string   `json:"eta_text,omitempty"`
	Steps   []string `json:"steps,omitempty"`
}

type researchStatusResponse struct {
	ID     string `json:"id"`
	ChatID string `json:"chat_id"`
	State  string `json:"state"`
}

type researchSource struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type researchResultResponse struct {
	ID      string           `json:"id"`
	ChatID  string           `json:"chat_id"`
	Text    string           `json:"text"`
	Sources []researchSource `json:"sources,omitempty"`
}

type researchResourceResponse struct {
	ID      string                  `json:"id"`
	ChatID  string                  `json:"chat_id"`
	State   string                  `json:"state"`
	Title   string                  `json:"title,omitempty"`
	ETAText string                  `json:"eta_text,omitempty"`
	Steps   []string                `json:"steps,omitempty"`
	Result  *researchResultResponse `json:"result"`
}

func (s *Server) handleResearchCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var req researchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	model := s.resolveModel(req.Model)
	if model == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("model %q not found", req.Model))
		return
	}

	plan, err := s.client.CreateAndStartDeepResearch(r.Context(), req.Prompt, model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := researchCreateResponse{
		ID:      plan.Cid,
		ChatID:  plan.Cid,
		Title:   plan.Title,
		ETAText: plan.ETAText,
		Steps:   plan.Steps,
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleResearchStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "research id is required")
		return
	}

	status, err := s.client.CheckDeepResearch(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := researchStatusResponse{
		ID:     id,
		ChatID: id,
		State:  status.State,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleResearchGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "research id is required")
		return
	}

	status, err := s.client.CheckDeepResearch(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := researchResourceResponse{
		ID:     id,
		ChatID: id,
		State:  status.State,
	}
	if status.State == "done" {
		result, err := s.researchResult(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		resp.Result = result
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleResearchResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "research id is required")
		return
	}

	resp, err := s.researchResult(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) researchResult(ctx context.Context, id string) (*researchResultResponse, error) {
	text, sources, err := s.client.GetDeepResearchResult(ctx, id)
	if err != nil {
		return nil, err
	}

	keys := make([]int, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	respSources := make([]researchSource, 0, len(sources))
	for _, key := range keys {
		s := sources[key]
		respSources = append(respSources, researchSource{
			URL:   s.URL,
			Title: s.Title,
		})
	}

	return &researchResultResponse{
		ID:      id,
		ChatID:  id,
		Text:    text,
		Sources: respSources,
	}, nil
}
