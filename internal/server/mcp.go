package server

import (
	"net/http"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (s *Server) buildMCPHandler() http.Handler {
	mcpServer := mcpserver.NewMCPServer("gemini-web-cli", "dev",
		mcpserver.WithToolCapabilities(false),
		// Advertise resources and prompts capabilities so MCP clients that probe
		// resources/list or prompts/list during connect get an empty result
		// instead of JSON-RPC -32601 "method not found". We register none,
		// so the handlers return empty lists.
		mcpserver.WithResourceCapabilities(false, false),
		mcpserver.WithPromptCapabilities(false),
		mcpserver.WithHooks(mcpHooks()),
		mcpserver.WithToolHandlerMiddleware(mcpToolLoggingMiddleware),
	)
	s.registerMCPTools(mcpServer)
	return mcpLoggingMiddleware(mcpserver.NewStreamableHTTPServer(mcpServer,
		mcpserver.WithStateLess(true),
	))
}
