package server

import (
	"net/http"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (s *Server) buildMCPHandler() http.Handler {
	mcpServer := mcpserver.NewMCPServer("gemini-web-cli", "dev",
		mcpserver.WithToolCapabilities(false),
	)
	s.registerMCPTools(mcpServer)
	return mcpserver.NewStreamableHTTPServer(mcpServer,
		mcpserver.WithStateLess(true),
	)
}
