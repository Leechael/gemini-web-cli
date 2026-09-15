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
    "/v1/accounts": {
      "get": {
        "operationId": "listAccounts",
        "summary": "List cookie accounts and login health",
        "responses": {
          "200": {
            "description": "Account list",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/AccountList" }
              }
            }
          }
        }
      }
    },
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
    "/v1/notebooks": {
      "post": {
        "operationId": "createNotebook",
        "summary": "Create a notebook",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["title"],
                "properties": { "title": { "type": "string" } }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Notebook created (resource name and title)" }
        }
      }
    },
    "/v1/notebooks/{id}": {
      "get": {
        "operationId": "getNotebook",
        "summary": "Get a notebook's title, emoji, and source list",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" }, "description": "Notebook uuid (without the notebooks/ prefix)" }
        ],
        "responses": {
          "200": { "description": "Notebook details" },
          "404": { "description": "Notebook not found" }
        }
      }
    },
    "/v1/notebooks/{id}/chats": {
      "get": {
        "operationId": "listNotebookChats",
        "summary": "List chats inside a notebook",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Chat list (newest first)" }
        }
      }
    },
    "/v1/notebooks/{id}/sources": {
      "post": {
        "operationId": "addNotebookSource",
        "summary": "Attach a source to a notebook (local file path on the serve host, or a web URL)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "path": { "type": "string", "description": "Local file path to upload (on the serve host)" },
                  "url": { "type": "string", "description": "http(s) URL to attach" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Updated notebook with source list" }
        }
      }
    },
    "/v1/notebooks/{id}/sources/{sid}": {
      "delete": {
        "operationId": "removeNotebookSource",
        "summary": "Remove a source from a notebook",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "sid", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Source removed" }
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
    "/v1/research/{id}": {
      "get": {
        "operationId": "getResearch",
        "summary": "Get deep research resource state and latest result",
        "parameters": [{
          "name": "id",
          "in": "path",
          "required": true,
          "schema": { "type": "string" }
        }],
        "responses": {
          "200": {
            "description": "Research resource",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/ResearchResourceResponse" }
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
      "AccountList": {
        "type": "object",
        "properties": {
          "object": { "type": "string", "enum": ["list"] },
          "accounts": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/AccountStatus" }
          }
        }
      },
      "AccountStatus": {
        "type": "object",
        "properties": {
          "index": { "type": "integer", "description": "1-based account index" },
          "name": { "type": "string", "description": "Cookie filename only, e.g. alice.json" },
          "logged_in": { "type": "boolean" },
          "last_error": { "type": "string" },
          "last_error_at": { "type": "string", "format": "date-time" }
        }
      },
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
          "model": { "type": "string", "description": "Model name (e.g. gemini-3.8-flash). Defaults to auto-select." },
          "messages": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/ChatMessage" }
          },
          "stream": { "type": "boolean", "default": false },
          "chat_id": { "type": "string", "description": "gemini-web-cli extension: continue an existing Gemini chat by ID. If omitted, the server uses chat-map state when available or starts a new Gemini chat." },
          "notebook": { "type": "string", "description": "gemini-web-cli extension: scope the request to a notebook (id with or without the notebooks/ prefix)." }
        }
      },
      "ChatMessage": {
        "type": "object",
        "required": ["role", "content"],
        "properties": {
          "role": { "type": "string", "enum": ["system", "developer", "user", "assistant"] },
          "content": {
            "oneOf": [
              { "type": "string" },
              {
                "type": "array",
                "items": { "$ref": "#/components/schemas/ChatContentPart" }
              }
            ]
          },
          "reasoning_content": { "type": "string" }
        }
      },
      "ChatContentPart": {
        "type": "object",
        "required": ["type", "text"],
        "properties": {
          "type": { "type": "string", "enum": ["text"] },
          "text": { "type": "string" }
        }
      },
      "ChatCompletionResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "chat_id": { "type": "string", "description": "gemini-web-cli extension: Gemini chat ID for continuation" },
          "object": { "type": "string" },
          "created": { "type": "integer" },
          "model": { "type": "string" },
          "choices": {
            "type": "array",
            "items": { "$ref": "#/components/schemas/ChatChoice" }
          },
          "usage": { "$ref": "#/components/schemas/ChatUsage" }
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
      "ChatUsage": {
        "type": "object",
        "properties": {
          "prompt_tokens": { "type": "integer" },
          "completion_tokens": { "type": "integer" },
          "total_tokens": { "type": "integer" }
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
          "chat_id": { "type": "string" },
          "title": { "type": "string" },
          "eta_text": { "type": "string" },
          "steps": { "type": "array", "items": { "type": "string" } }
        }
      },
      "ResearchStatusResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "chat_id": { "type": "string" },
          "state": { "type": "string", "enum": ["done", "running", "pending_confirm", "not_research", "empty"] }
        }
      },
      "ResearchResourceResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "chat_id": { "type": "string" },
          "state": { "type": "string", "enum": ["done", "running", "pending_confirm", "not_research", "empty"] },
          "title": { "type": "string" },
          "eta_text": { "type": "string" },
          "steps": { "type": "array", "items": { "type": "string" } },
          "result": {
            "oneOf": [
              { "$ref": "#/components/schemas/ResearchResultResponse" },
              { "type": "null" }
            ]
          }
        }
      },
      "ResearchResultResponse": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "chat_id": { "type": "string" },
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
