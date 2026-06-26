package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

type Server struct {
	client *client.Client
	cfg    client.Config
	mux    *http.ServeMux

	stopRefresh context.CancelFunc

	mu             sync.RWMutex
	modelRegistry  []*types.Model
	cachedModelsAt time.Time
}

func New(cfg client.Config) (*Server, error) {
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}

	s := &Server{
		client: c,
		cfg:    cfg,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) Init(ctx context.Context) error {
	if err := s.client.Init(ctx); err != nil {
		return err
	}

	refreshCtx, cancel := context.WithCancel(context.Background())
	s.stopRefresh = cancel
	go s.refreshLoop(refreshCtx)

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Close() {
	if s.stopRefresh != nil {
		s.stopRefresh()
	}
	s.client.Close()
}

func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s,
	}
	log.Printf("listening on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /v1/models", s.handleModels)
	s.mux.HandleFunc("POST /v1/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("POST /v1/research", s.handleResearchCreate)
	s.mux.HandleFunc("GET /v1/research/{id}/status", s.handleResearchStatus)
	s.mux.HandleFunc("GET /v1/research/{id}/result", s.handleResearchResult)
}

func (s *Server) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.client.Init(ctx); err != nil {
				log.Printf("token refresh failed: %v", err)
			} else {
				log.Printf("token refreshed")
			}
		}
	}
}

func (s *Server) resolveModel(name string) *types.Model {
	if name == "" || name == "auto" {
		return types.FindModel("unspecified")
	}
	if m := types.FindModel(name); m != nil {
		return m
	}
	return nil
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":{"message":%q,"type":"invalid_request_error","code":%q}}`, message, http.StatusText(code))
}
