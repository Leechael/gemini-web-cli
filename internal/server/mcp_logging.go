package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	mcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// maxLoggedArgLen caps a single string argument when rendering tool call args
// for the log line. Long prompts and research texts would otherwise dominate
// server logs.
const maxLoggedArgLen = 120

// mcpCallStartKey carries the per-tool-call start time from the tool handler
// middleware to the after-call hook.
type mcpCallStartKey struct{}

// mcpLoggingMiddleware wraps the MCP HTTP handler and logs one line per
// inbound HTTP request: JSON-RPC method, HTTP status, and duration. It covers
// every transport-level call (initialize, ping, tools/list, tools/call, ...).
func mcpLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		method := mcpMethodFromRequest(r)
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		log.Printf("mcp http %s status=%d dur=%s", method, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

// statusWriter captures the response status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusWriter) WriteHeader(code int) {
	if s.wroteHeader {
		return
	}
	s.status = code
	s.wroteHeader = true
	s.ResponseWriter.WriteHeader(code)
}

// mcpMethodFromRequest extracts the JSON-RPC method from the request body for
// logging without consuming it. It re-arms r.Body so downstream handlers read
// the original payload. On any parse failure it returns "?".
func mcpMethodFromRequest(r *http.Request) string {
	if r.Body == nil {
		return "?"
	}
	buf, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		r.Body = http.NoBody
		return "?"
	}
	r.Body = io.NopCloser(strings.NewReader(string(buf)))

	var peek struct {
		Method string `json:"method"`
	}
	if json.Unmarshal(buf, &peek) != nil || peek.Method == "" {
		return "?"
	}
	return peek.Method
}

// mcpHooks returns the *mcpserver.Hooks that log MCP connections and tool
// call starts. The matching completion log (with duration) is emitted by
// mcpToolLoggingMiddleware, which shares the request ctx and can measure the
// actual handler duration.
func mcpHooks() *mcpserver.Hooks {
	hooks := &mcpserver.Hooks{}

	hooks.AddAfterInitialize(func(ctx context.Context, id any, req *mcp.InitializeRequest, res *mcp.InitializeResult) {
		client := "?"
		if v := req.Params.ClientInfo.Name; v != "" {
			client = v
			if req.Params.ClientInfo.Version != "" {
				client += "/" + req.Params.ClientInfo.Version
			}
		}
		log.Printf("mcp connect client=%q protocol=%q", client, req.Params.ProtocolVersion)
	})

	hooks.AddBeforeCallTool(func(ctx context.Context, id any, req *mcp.CallToolRequest) {
		log.Printf("mcp call start tool=%q args=%s", req.Params.Name, summarizeMCPArgs(req.GetArguments()))
	})

	return hooks
}

// mcpToolLoggingMiddleware is a ToolHandlerMiddleware that records the call
// start on the ctx and logs completion with duration and error state. The
// start time is also readable by any after-call hook via mcpCallStartKey.
func mcpToolLoggingMiddleware(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		ctx = context.WithValue(ctx, mcpCallStartKey{}, start)
		result, err := next(ctx, req)
		dur := time.Since(start).Round(time.Millisecond)
		if err != nil {
			log.Printf("mcp call done tool=%q error=%q dur=%s", req.Params.Name, err.Error(), dur)
			return result, err
		}
		isError := result != nil && result.IsError
		log.Printf("mcp call done tool=%q is_error=%t dur=%s", req.Params.Name, isError, dur)
		return result, nil
	}
}

// summarizeMCPArgs renders a tool call's arguments as compact JSON, truncating
// long string values so logs stay readable. It does not mutate the caller's
// map.
func summarizeMCPArgs(src map[string]any) string {
	if len(src) == 0 {
		return "{}"
	}
	trimmed := make(map[string]any, len(src))
	for k, v := range src {
		if s, ok := v.(string); ok && len(s) > maxLoggedArgLen {
			trimmed[k] = s[:maxLoggedArgLen] + "..."
		} else {
			trimmed[k] = v
		}
	}
	b, err := json.Marshal(trimmed)
	if err != nil {
		return "{<unmarshalable>}"
	}
	return string(b)
}
