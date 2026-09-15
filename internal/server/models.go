package server

import (
	"net/http"
	"time"

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

type accountStatusJSON struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	LoggedIn    bool   `json:"logged_in"`
	LastError   string `json:"last_error,omitempty"`
	LastErrorAt string `json:"last_error_at,omitempty"`
}

type accountListJSON struct {
	Object   string              `json:"object"`
	Accounts []accountStatusJSON `json:"accounts"`
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	var accounts []accountStatusJSON
	if s.pool != nil {
		for _, snap := range s.pool.AccountSnapshots() {
			item := accountStatusJSON{
				Index:     snap.Index,
				Name:      snap.Name,
				LoggedIn:  snap.LoggedIn,
				LastError: snap.LastError,
			}
			if !snap.LastErrorAt.IsZero() {
				item.LastErrorAt = snap.LastErrorAt.UTC().Format(time.RFC3339)
			}
			accounts = append(accounts, item)
		}
	}
	if accounts == nil {
		accounts = []accountStatusJSON{}
	}
	writeJSON(w, http.StatusOK, accountListJSON{Object: "list", Accounts: accounts})
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
	if s.pool != nil {
		if models := s.pool.AvailableModels(); len(models) > 0 {
			return models
		}
	}
	return types.Models
}
