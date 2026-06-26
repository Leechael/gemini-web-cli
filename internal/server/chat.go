package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	ChatID   string        `json:"chat_id,omitempty"`
}

type chatMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

func (m *chatMessage) UnmarshalJSON(data []byte) error {
	type plain struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	m.Role = p.Role

	if len(p.Content) == 0 || string(p.Content) == "null" {
		return nil
	}

	if p.Content[0] == '"' {
		return json.Unmarshal(p.Content, &m.Content)
	}

	if p.Content[0] == '[' {
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(p.Content, &parts); err != nil {
			return err
		}
		var texts []string
		for _, part := range parts {
			if part.Type == "text" {
				texts = append(texts, part.Text)
			}
		}
		m.Content = strings.Join(texts, "\n")
		return nil
	}

	return json.Unmarshal(p.Content, &m.Content)
}

type chatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   *chatUsage   `json:"usage,omitempty"`
}

type chatChoice struct {
	Index        int          `json:"index"`
	Message      *chatMessage `json:"message,omitempty"`
	Delta        *chatMessage `json:"delta,omitempty"`
	FinishReason *string      `json:"finish_reason"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages is required")
		return
	}

	model := s.resolveModel(req.Model)
	if model == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("model %q not found", req.Model))
		return
	}

	last := req.Messages[len(req.Messages)-1]
	prompt := last.Content

	if len(req.Messages) > 1 && req.ChatID == "" {
		var parts []string
		for _, m := range req.Messages[:len(req.Messages)-1] {
			switch m.Role {
			case "system":
				parts = append(parts, fmt.Sprintf("[System]\n%s", m.Content))
			case "user":
				parts = append(parts, fmt.Sprintf("[User]\n%s", m.Content))
			case "assistant":
				parts = append(parts, fmt.Sprintf("[Assistant]\n%s", m.Content))
			}
		}
		if len(parts) > 0 {
			prompt = strings.Join(parts, "\n\n") + "\n\n[User]\n" + prompt
		}
	}

	ctx := r.Context()

	if req.ChatID != "" {
		latest, _ := s.client.FetchLatestChatResponse(ctx, req.ChatID)
		metadata := make([]string, 10)
		metadata[0] = req.ChatID
		if latest != nil {
			if latest.Rid != "" {
				metadata[1] = latest.Rid
			}
			if latest.RCid != "" {
				metadata[2] = latest.RCid
			}
		}

		if req.Stream {
			s.writeSSE(w, req.ChatID, model.Name, func(emit func(delta, reasoning string)) error {
				_, err := s.client.SendMessageStream(ctx, prompt, metadata, model, func(out *types.ModelOutput) {
					emit(out.TextDelta, out.ThoughtsDelta)
				})
				return err
			})
		} else {
			output, err := s.client.SendMessage(ctx, prompt, metadata, model)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.writeCompletion(w, req.ChatID, model.Name, output.Text, output.Thoughts)
		}
		return
	}

	if req.Stream {
		chatID := ""
		s.writeSSE(w, "", model.Name, func(emit func(delta, reasoning string)) error {
			_, err := s.client.GenerateContentStream(ctx, prompt, model, func(out *types.ModelOutput) {
				if chatID == "" && len(out.Metadata) > 0 {
					chatID = out.Metadata[0]
				}
				emit(out.TextDelta, out.ThoughtsDelta)
			})
			return err
		})
	} else {
		output, err := s.client.GenerateContent(ctx, prompt, model)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		chatID := ""
		if len(output.Metadata) > 0 {
			chatID = output.Metadata[0]
		}
		s.writeCompletion(w, chatID, model.Name, output.Text, output.Thoughts)
	}
}

func (s *Server) writeCompletion(w http.ResponseWriter, chatID, modelName, text, thoughts string) {
	stop := "stop"
	msg := &chatMessage{Role: "assistant", Content: text, ReasoningContent: thoughts}
	resp := chatCompletionResponse{
		ID:      "chatcmpl-" + chatID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []chatChoice{{
			Index:        0,
			Message:      msg,
			FinishReason: &stop,
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) writeSSE(w http.ResponseWriter, chatID, modelName string, generate func(emit func(delta, reasoning string)) error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	id := "chatcmpl-" + chatID

	emit := func(delta, reasoning string) {
		if delta == "" && reasoning == "" {
			return
		}
		chunk := chatCompletionResponse{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   modelName,
			Choices: []chatChoice{{
				Index: 0,
				Delta: &chatMessage{Role: "assistant", Content: delta, ReasoningContent: reasoning},
			}},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	if err := generate(emit); err != nil {
		fmt.Fprintf(w, "data: {\"error\":{\"message\":%q}}\n\n", err.Error())
		flusher.Flush()
		return
	}

	stop := "stop"
	finalChunk := chatCompletionResponse{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []chatChoice{{
			Index:        0,
			Delta:        &chatMessage{Role: "assistant"},
			FinishReason: &stop,
		}},
	}
	data, _ := json.Marshal(finalChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}
