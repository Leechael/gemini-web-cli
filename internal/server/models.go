package server

import (
	"net/http"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

type openAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type openAIModelList struct {
	Object string        `json:"object"`
	Data   []openAIModel `json:"data"`
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	var models []openAIModel
	for _, m := range s.availableModels() {
		if m.Name == "unspecified" {
			continue
		}
		models = append(models, openAIModel{
			ID:      m.Name,
			Object:  "model",
			Created: 0,
			OwnedBy: "google",
		})
	}

	writeJSON(w, http.StatusOK, openAIModelList{
		Object: "list",
		Data:   models,
	})
}

func (s *Server) availableModels() []types.Model {
	if s.client != nil {
		if models := s.client.AvailableModels(); len(models) > 0 {
			return models
		}
	}
	return types.Models
}
