package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	mcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// mcpLoggingMiddleware wraps the MCP HTTP handler and logs one line per
// inbound HTTP request: JSON-RPC method, HTTP status, and duration. It covers
// every transport-level call (initialize, ping, tools/list, tools/call, ...).
func mcpLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		if r.Method == http.MethodPost {
			if r.ContentLength > maxRequestBodyBytes {
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				logMCPHTTP(r, "POST?", http.StatusRequestEntityTooLarge, start)
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
			}
		}

		method, tooLarge := mcpMethodFromRequest(r)
		if tooLarge {
			http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
			logMCPHTTP(r, method, http.StatusRequestEntityTooLarge, start)
			return
		}
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		logMCPHTTP(r, method, sw.status, start)
	})
}

func logMCPHTTP(r *http.Request, method string, status int, start time.Time) {
	log.Printf("mcp http %s path=%s remote=%s status=%d dur=%s",
		method, r.URL.Path, clientIP(r), status, time.Since(start).Round(time.Millisecond))
}

// clientIP returns the TCP peer IP without the port. It intentionally ignores
// X-Forwarded-For because serve can be exposed directly on a LAN and clients
// can spoof that header unless a trusted proxy boundary is configured.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// statusWriter captures the response status code while preserving optional
// streaming support expected by MCP transports.
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

func (s *statusWriter) Flush() {
	if flusher, ok := s.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// mcpMethodFromRequest extracts the JSON-RPC method from the request body
// for logging without consuming it. It re-arms r.Body so downstream handlers
// read the original payload. For non-POST requests (GET SSE probe, DELETE
// session terminate, OPTIONS preflight) the body is empty, so it returns the
// HTTP method to keep the log line meaningful. On POST parse failure it
// returns "POST?".
func mcpMethodFromRequest(r *http.Request) (string, bool) {
	if r.Method != http.MethodPost {
		return r.Method, false
	}
	if r.Body == nil {
		return "POST?", false
	}
	buf, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		r.Body = http.NoBody
		var maxErr *http.MaxBytesError
		return "POST?", errors.As(err, &maxErr)
	}
	r.Body = io.NopCloser(bytes.NewReader(buf))

	// JSON-RPC over HTTP may carry a single object or a batch array. Extract
	// the first method found either way.
	if method := methodFromJSONRPC(buf); method != "" {
		return method, false
	}
	return "POST?", false
}

// methodFromJSONRPC returns the JSON-RPC method from a single-object body,
// or the first method from a batch array body. Empty on any failure.
func methodFromJSONRPC(buf []byte) string {
	trimmed := bytes.TrimSpace(buf)
	if len(trimmed) == 0 {
		return ""
	}
	if trimmed[0] == '[' {
		var batch []struct {
			Method string `json:"method"`
		}
		if json.Unmarshal(trimmed, &batch) != nil || len(batch) == 0 {
			return ""
		}
		return batch[0].Method
	}
	var single struct {
		Method string `json:"method"`
	}
	if json.Unmarshal(trimmed, &single) != nil {
		return ""
	}
	return single.Method
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

// mcpToolLoggingMiddleware is a ToolHandlerMiddleware that logs completion
// with duration and error state.
func mcpToolLoggingMiddleware(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
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

// summarizeMCPArgs renders a tool call's argument shape as compact JSON.
// It deliberately avoids logging string values: prompts often contain user
// secrets or private documents. Scalars that are not strings are kept because
// they are useful for debugging pagination and toggles.
func summarizeMCPArgs(src map[string]any) string {
	if len(src) == 0 {
		return "{}"
	}
	summary := make(map[string]any, len(src))
	for k, v := range src {
		summary[k] = summarizeMCPArgValue(v)
	}
	b, err := json.Marshal(summary)
	if err != nil {
		return "{<unmarshalable>}"
	}
	return string(b)
}

func summarizeMCPArgValue(v any) any {
	switch value := v.(type) {
	case string:
		return fmt.Sprintf("string(len=%d)", len(value))
	case []any:
		return fmt.Sprintf("array(len=%d)", len(value))
	case map[string]any:
		return fmt.Sprintf("object(keys=%d)", len(value))
	default:
		return value
	}
}
