package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
	MCPDefaultModel string
}

type Server struct {
	client          *client.Client
	mux             *http.ServeMux
	apiKey          string
	exposeThoughts  bool
	mcpDefaultModel string
	stateInfo       StateInfo
	chatMap         *serverstate.ChatMapStore

	stopRefresh context.CancelFunc
}

func New(cfg client.Config, apiKey string, exposeThoughts bool, mcpDefaultModel string, stateInfo StateInfo) (*Server, error) {
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
		client:          c,
		mux:             http.NewServeMux(),
		apiKey:          apiKey,
		exposeThoughts:  exposeThoughts,
		mcpDefaultModel: mcpDefaultModel,
		stateInfo:       stateInfo,
		chatMap:         chatMap,
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

// bannerHost returns the host to display in startup URLs. When the server is
// bound to all interfaces (0.0.0.0 or ::), it sniffs the host's non-loopback
// IPv4 addresses so the printed URLs are actually reachable from the LAN.
// Multiple interfaces (e.g. Wi-Fi + Ethernet) yield multiple printed URLs.
// For a specific bind host it returns that host unchanged.
func bannerHost(addr string) []string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host != "" && host != "0.0.0.0" && host != "::" && host != "[::]" {
		return []string{host}
	}
	return lanIPv4s()
}

// lanIPv4s returns non-loopback, non-link-local IPv4 addresses of the host.
// It returns nil if none are found, in which case callers fall back to
// 127.0.0.1.
func lanIPv4s() []string {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var addrs []string
	for _, ifi := range ifs {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		ipAddrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range ipAddrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			addrs = append(addrs, ip.String())
		}
	}
	return addrs
}

func printBanner(addr string, stateInfo StateInfo) {
	hosts := bannerHost(addr)
	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1"}
	}
	// For banner URLs, prefer the first LAN IP when bound to 0.0.0.0, but
	// also list alternates so multi-interface (Wi-Fi + Ethernet) setups are
	// visible. Port is reused from the bind addr.
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = "8080"
	}
	bases := make([]string, 0, len(hosts))
	for _, h := range hosts {
		bases = append(bases, "http://"+net.JoinHostPort(h, port))
	}
	primary := bases[0]

	w := os.Stderr
	fmt.Fprintf(w, "gemini-web-cli server running on %s\n", primary)
	if len(bases) > 1 {
		fmt.Fprintf(w, "  also reachable on:\n")
		for _, b := range bases[1:] {
			fmt.Fprintf(w, "    %s\n", b)
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "OpenAI-compatible API:\n")
	fmt.Fprintf(w, "  export OPENAI_API_BASE=%s/v1\n\n", primary)
	fmt.Fprintf(w, "  # Chat (streaming)\n")
	fmt.Fprintf(w, "  curl %s/v1/chat/completions \\\n", primary)
	fmt.Fprintf(w, "    -H 'Content-Type: application/json' \\\n")
	fmt.Fprintf(w, "    -d '{\"model\":\"gemini-3.5-flash\",\"stream\":true,\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}'\n\n")
	fmt.Fprintf(w, "  # Chat (non-streaming)\n")
	fmt.Fprintf(w, "  curl %s/v1/chat/completions \\\n", primary)
	fmt.Fprintf(w, "    -H 'Content-Type: application/json' \\\n")
	fmt.Fprintf(w, "    -d '{\"model\":\"gemini-3.5-flash\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}'\n\n")
	fmt.Fprintf(w, "  # List models\n")
	fmt.Fprintf(w, "  curl %s/v1/models\n\n", primary)
	fmt.Fprintf(w, "Deep Research API:\n")
	fmt.Fprintf(w, "  # Submit\n")
	fmt.Fprintf(w, "  curl -X POST %s/v1/research \\\n", primary)
	fmt.Fprintf(w, "    -H 'Content-Type: application/json' \\\n")
	fmt.Fprintf(w, "    -d '{\"prompt\":\"Research topic here\"}'\n\n")
	fmt.Fprintf(w, "  # Check state / status / result\n")
	fmt.Fprintf(w, "  curl %s/v1/research/{id}\n", primary)
	fmt.Fprintf(w, "  curl %s/v1/research/{id}/status\n", primary)
	fmt.Fprintf(w, "  curl %s/v1/research/{id}/result\n\n", primary)
	fmt.Fprintf(w, "MCP Server:\n")
	fmt.Fprintf(w, "  %s/mcp\n", primary)
	if stateInfo.MCPDefaultModel != "" {
		fmt.Fprintf(w, "  default model: %s\n", stateInfo.MCPDefaultModel)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Docs:\n")
	fmt.Fprintf(w, "  Swagger UI:   %s/docs\n", primary)
	fmt.Fprintf(w, "  OpenAPI spec: %s/openapi.json\n\n", primary)
	fmt.Fprintf(w, "State:\n")
	fmt.Fprintf(w, "  state_dir: %s\n", stateValue(stateInfo.StateDir, "<none>"))
	fmt.Fprintf(w, "  cookies: %s\n", stateValue(stateInfo.CookieSource, "<none>"))
	fmt.Fprintf(w, "  chat_mapping: %s\n\n", stateValue(stateInfo.ChatMappingMode, "memory only"))
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
	s.mux.Handle("/mcp", s.buildMCPHandler())
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
