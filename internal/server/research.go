package server

import (
	"encoding/json"
	"net/http"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

type researchRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model,omitempty"`
}

type researchCreateResponse struct {
	ID      string   `json:"id"`
	Title   string   `json:"title,omitempty"`
	ETAText string   `json:"eta_text,omitempty"`
	Steps   []string `json:"steps,omitempty"`
}

type researchStatusResponse struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

type researchSource struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type researchResultResponse struct {
	ID      string           `json:"id"`
	Text    string           `json:"text"`
	Sources []researchSource `json:"sources,omitempty"`
}

func (s *Server) handleResearchCreate(w http.ResponseWriter, r *http.Request) {
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
		model = types.FindModel("unspecified")
	}

	plan, err := s.client.CreateAndStartDeepResearch(r.Context(), req.Prompt, model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := researchCreateResponse{
		ID:      plan.Cid,
		Title:   plan.Title,
		ETAText: plan.ETAText,
		Steps:   plan.Steps,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
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
		ID:    id,
		State: status.State,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleResearchResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "research id is required")
		return
	}

	text, sources, err := s.client.GetDeepResearchResult(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var respSources []researchSource
	for i := 0; i < len(sources); i++ {
		s, ok := sources[i]
		if !ok {
			continue
		}
		respSources = append(respSources, researchSource{
			URL:   s.URL,
			Title: s.Title,
		})
	}

	resp := researchResultResponse{
		ID:      id,
		Text:    text,
		Sources: respSources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
