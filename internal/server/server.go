package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client"
	serverstate "github.com/Leechael/gemini-web-cli/internal/server/state"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const maxRequestBodyBytes = 1 << 20

type StateInfo struct {
	StateDir        string
	CookieSource    string
	ChatMappingPath string
	ChatMappingMode string
}

type Server struct {
	client         *client.Client
	mux            *http.ServeMux
	apiKey         string
	exposeThoughts bool
	stateInfo      StateInfo
	chatMap        *serverstate.ChatMapStore

	stopRefresh context.CancelFunc
}

func New(cfg client.Config, apiKey string, exposeThoughts bool, stateInfo StateInfo) (*Server, error) {
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	chatMap, err := serverstate.NewChatMapStore(stateInfo.ChatMappingPath)
	if err != nil {
		c.Close()
		return nil, err
	}

	s := &Server{
		client:         c,
		mux:            http.NewServeMux(),
		apiKey:         apiKey,
		exposeThoughts: exposeThoughts,
		stateInfo:      stateInfo,
		chatMap:        chatMap,
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) Init(ctx context.Context) error {
	if err := s.client.Init(ctx); err != nil {
		return err
	}
	if err := s.client.FetchAndCacheModels(ctx); err != nil {
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
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	printBanner(addr, s.stateInfo)
	return srv.ListenAndServe()
}

func printBanner(addr string, stateInfo StateInfo) {
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
	fmt.Printf("  # Check state / status / result\n")
	fmt.Printf("  curl %s/v1/research/{id}\n", base)
	fmt.Printf("  curl %s/v1/research/{id}/status\n", base)
	fmt.Printf("  curl %s/v1/research/{id}/result\n\n", base)
	fmt.Printf("Docs:\n")
	fmt.Printf("  Swagger UI:   %s/docs\n", base)
	fmt.Printf("  OpenAPI spec: %s/openapi.json\n\n", base)
	fmt.Printf("State:\n")
	fmt.Printf("  state_dir: %s\n", stateValue(stateInfo.StateDir, "<none>"))
	fmt.Printf("  cookies: %s\n", stateValue(stateInfo.CookieSource, "<none>"))
	fmt.Printf("  chat_mapping: %s\n\n", stateValue(stateInfo.ChatMappingMode, "memory only"))
}

func stateValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /v1/models", s.requireAuth(s.handleModels))
	s.mux.HandleFunc("POST /v1/chat/completions", s.requireAuth(s.handleChatCompletions))
	s.mux.HandleFunc("POST /v1/research", s.requireAuth(s.handleResearchCreate))
	s.mux.HandleFunc("GET /v1/research/{id}", s.requireAuth(s.handleResearchGet))
	s.mux.HandleFunc("GET /v1/research/{id}/status", s.requireAuth(s.handleResearchStatus))
	s.mux.HandleFunc("GET /v1/research/{id}/result", s.requireAuth(s.handleResearchResult))
	s.mux.HandleFunc("GET /openapi.json", s.handleOpenAPISpec)
	s.mux.HandleFunc("GET /docs", s.handleSwaggerUI)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" {
			next(w, r)
			return
		}
		if constantTimeEqual(apiKeyFromRequest(r), s.apiKey) {
			next(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, "missing or invalid API key")
	}
}

func apiKeyFromRequest(r *http.Request) string {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func constantTimeEqual(a, b string) bool {
	if a == "" || b == "" || len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
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
	if s.client != nil {
		if m := s.client.ResolveModel(name); m != nil {
			return m
		}
	}
	if m := types.FindModel(name); m != nil {
		return m
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := fmt.Fprintf(w, `{"error":{"message":%q,"type":"invalid_request_error","code":%q}}`, message, http.StatusText(code)); err != nil {
		log.Printf("write error response failed: %v", err)
	}
}
