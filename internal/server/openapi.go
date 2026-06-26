package server

import (
	"fmt"
	"net/http"
)

const openapiSpec = `{
  "openapi": "3.1.0",
  "info": {
    "title": "gemini-web-cli",
    "description": "OpenAI-compatible API proxy for Google Gemini",
    "version": "1.0.0"
  },
  "servers": [{"url": "/"}],
  "paths": {
    "/v1/models": {
      "get": {
        "operationId": "listModels",
        "summary": "List available models",
        "responses": {
          "200": {
            "description": "Model list",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ModelList" }
              }
            }
          }
        }
      }
    },
    "/v1/chat/completions": {
      "post": {
        "operationId": "createChatCompletion",
        "summary": "Create a chat completion (OpenAI-compatible)",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/ChatCompletionRequest" }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Chat completion response or SSE stream",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ChatCompletionResponse" }
              },
              "text/event-stream": {
                "description": "SSE stream when stream=true"
              }
            }
          }
        }
      }
    },
    "/v1/research": {
      "post": {
        "operationId": "createResearch",
        "summary": "Submit a deep research task",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/ResearchRequest" }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Research task created",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ResearchCreateResponse" }
              }
            }
          }
        }
      }
    },
    "/v1/research/{id}/status": {
      "get": {
        "operationId": "getResearchStatus",
        "summary": "Check deep research progress",
        "parameters": [{
          "name": "id",
          "in": "path",
          "required": true,
          "schema": { "type": "string" }
        }],
        "responses": {
          "200": {
            "description": "Research status",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ResearchStatusResponse" }
              }
            }
          }
        }
      }
    },
    "/v1/research/{id}/result": {
      "get": {
        "operationId": "getResearchResult",
        "summary": "Get deep research result",
        "parameters": [{
          "name": "id",
          "in": "path",
          "required": true,
          "schema": { "type": "string" }
        }],
        "responses": {
          "200": {
            "description": "Research result",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ResearchResultResponse" }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "ModelList": {
        "type": "object",
        "properties": {
          "object": { "type": "string", "enum": ["list"] },
          "data": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/Model" }
          }
        }
      },
      "Model": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "object": { "type": "string", "enum": ["model"] },
          "created": { "type": "integer" },
          "owned_by": { "type": "string" }
        }
      },
      "ChatCompletionRequest": {
        "type": "object",
        "required": ["messages"],
        "properties": {
          "model": { "type": "string", "description": "Model name (e.g. gemini-3.5-flash). Defaults to auto-select." },
          "messages": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/ChatMessage" }
          },
          "stream": { "type": "boolean", "default": false },
          "chat_id": { "type": "string", "description": "Continue an existing chat by ID" }
        }
      },
      "ChatMessage": {
        "type": "object",
        "required": ["role", "content"],
        "properties": {
          "role": { "type": "string", "enum": ["system", "user", "assistant"] },
          "content": { "type": "string" }
        }
      },
      "ChatCompletionResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "object": { "type": "string" },
          "created": { "type": "integer" },
          "model": { "type": "string" },
          "choices": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/ChatChoice" }
          }
        }
      },
      "ChatChoice": {
        "type": "object",
        "properties": {
          "index": { "type": "integer" },
          "message": { "$ref": "#/components/schemas/ChatMessage" },
          "delta": { "$ref": "#/components/schemas/ChatMessage" },
          "finish_reason": { "type": "string", "nullable": true }
        }
      },
      "ResearchRequest": {
        "type": "object",
        "required": ["prompt"],
        "properties": {
          "prompt": { "type": "string" },
          "model": { "type": "string" }
        }
      },
      "ResearchCreateResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "title": { "type": "string" },
          "eta_text": { "type": "string" },
          "steps": { "type": "array", "items": { "type": "string" } }
        }
      },
      "ResearchStatusResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "state": { "type": "string", "enum": ["done", "running", "pending_confirm", "not_research", "empty"] }
        }
      },
      "ResearchResultResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "text": { "type": "string" },
          "sources": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "url": { "type": "string" },
                "title": { "type": "string" }
              }
            }
          }
        }
      }
    }
  }
}`

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>gemini-web-cli API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'BaseLayout',
    });
  </script>
</body>
</html>`

func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, openapiSpec)
}

func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, swaggerHTML)
}
