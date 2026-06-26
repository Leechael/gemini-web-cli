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
	printBanner(addr)
	return srv.ListenAndServe()
}

func printBanner(addr string) {
	base := "http://" + addr
	fmt.Printf("gemini-web-cli server running on %s\n\n", base)
	fmt.Printf("OpenAI-compatible API:\n")
	fmt.Printf("  export OPENAI_API_BASE=%s/v1\n\n", base)
	fmt.Printf("  # Chat (streaming)\n")
	fmt.Printf("  curl %s/v1/chat/completions \\\n", base)
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"model\":\"gemini-3.5-flash\",\"stream\":true,\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}'\n\n")
	fmt.Printf("  # Chat (non-streaming)\n")
	fmt.Printf("  curl %s/v1/chat/completions \\\n", base)
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"model\":\"gemini-3.5-flash\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}'\n\n")
	fmt.Printf("  # List models\n")
	fmt.Printf("  curl %s/v1/models\n\n", base)
	fmt.Printf("Deep Research API:\n")
	fmt.Printf("  # Submit\n")
	fmt.Printf("  curl -X POST %s/v1/research \\\n", base)
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"prompt\":\"Research topic here\"}'\n\n")
	fmt.Printf("  # Check status / Get result\n")
	fmt.Printf("  curl %s/v1/research/{id}/status\n", base)
	fmt.Printf("  curl %s/v1/research/{id}/result\n\n", base)
	fmt.Printf("Docs:\n")
	fmt.Printf("  Swagger UI:   %s/docs\n", base)
	fmt.Printf("  OpenAPI spec: %s/openapi.json\n\n", base)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /v1/models", s.handleModels)
	s.mux.HandleFunc("POST /v1/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("POST /v1/research", s.handleResearchCreate)
	s.mux.HandleFunc("GET /v1/research/{id}/status", s.handleResearchStatus)
	s.mux.HandleFunc("GET /v1/research/{id}/result", s.handleResearchResult)
	s.mux.HandleFunc("GET /openapi.json", s.handleOpenAPISpec)
	s.mux.HandleFunc("GET /docs", s.handleSwaggerUI)
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
