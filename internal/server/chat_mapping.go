package server

import (
	"fmt"
	"time"

	serverstate "github.com/Leechael/gemini-web-cli/internal/server/state"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

type mappedChatPlan struct {
	Prompt   string
	Metadata []string
	Source   string
}

func (s *Server) planMappedChat(messages []chatMessage) (mappedChatPlan, error) {
	prefixes, err := completedChatPrefixes(messages)
	if err != nil {
		return mappedChatPlan{}, err
	}
	for i := len(prefixes) - 1; i >= 0; i-- {
		prefix := prefixes[i]
		entry, ok := s.chatMap.Lookup(prefix.Hash)
		if !ok {
			continue
		}
		count := int(entry.MessageCount)
		if count == 0 {
			count = int(prefix.MessageCount)
		}
		if count > len(messages) {
			continue
		}
		suffix := messages[count:]
		if len(suffix) == 0 {
			return mappedChatPlan{}, fmt.Errorf("messages do not include a new user message after mapped chat state")
		}
		prompt, err := promptForMappedSuffix(entry, suffix)
		if err != nil {
			return mappedChatPlan{}, err
		}
		return mappedChatPlan{Prompt: prompt, Metadata: metadataFromEntry(entry), Source: chatPlanSource(entry)}, nil
	}

	prompt, err := flattenChatMessages(messages)
	if err != nil {
		return mappedChatPlan{}, err
	}
	return mappedChatPlan{Prompt: prompt, Source: "new"}, nil
}

func chatPlanSource(entry *serverstate.ChatMapEntry) string {
	if entry.Confidence == serverstate.ChatMapConfidence_VERIFIED {
		return "mapped_verified"
	}
	if entry.Confidence == serverstate.ChatMapConfidence_SYNTHETIC {
		return "mapped_synthetic"
	}
	return "mapped"
}

func promptForMappedSuffix(entry *serverstate.ChatMapEntry, suffix []chatMessage) (string, error) {
	if entry.Confidence == serverstate.ChatMapConfidence_VERIFIED && len(suffix) == 1 {
		role, content, err := canonicalChatMessage(suffix[0])
		if err != nil {
			return "", err
		}
		if role == "user" {
			return content, nil
		}
	}
	return flattenChatMessages(suffix)
}

func metadataFromEntry(entry *serverstate.ChatMapEntry) []string {
	metadata := make([]string, 10)
	metadata[0] = entry.ChatId
	metadata[1] = entry.Rid
	metadata[2] = entry.Rcid
	metadata[9] = entry.Context
	return metadata
}

func (s *Server) saveChatMapping(requestMessages []chatMessage, output *types.ModelOutput) error {
	if s.chatMap == nil || output == nil {
		return nil
	}
	chatID, rid, rcid, context := metadataFromOutput(output)
	if chatID == "" {
		return nil
	}
	assistant := chatMessage{Role: "assistant", Content: output.Text}
	completedMessages := append(append([]chatMessage{}, requestMessages...), assistant)
	completedRoot, ok, err := completedChatRoot(completedMessages)
	if err != nil || !ok {
		return err
	}

	now := time.Now().Unix()
	entries := []*serverstate.ChatMapEntry{
		{
			RootHash:     completedRoot.Hash,
			ParentHash:   completedRoot.ParentHash,
			ChatId:       chatID,
			Rid:          rid,
			Rcid:         rcid,
			Context:      context,
			UpdatedAt:    now,
			MessageCount: completedRoot.MessageCount,
			Confidence:   serverstate.ChatMapConfidence_VERIFIED,
		},
	}

	prefixes, err := completedChatPrefixes(requestMessages)
	if err != nil {
		return err
	}
	for _, prefix := range prefixes {
		entries = append(entries, &serverstate.ChatMapEntry{
			RootHash:     prefix.Hash,
			ParentHash:   prefix.ParentHash,
			ChatId:       chatID,
			Rid:          rid,
			Rcid:         rcid,
			Context:      context,
			UpdatedAt:    now,
			MessageCount: prefix.MessageCount,
			Confidence:   serverstate.ChatMapConfidence_SYNTHETIC,
		})
	}
	return s.chatMap.UpsertMany(entries)
}

func metadataFromOutput(output *types.ModelOutput) (chatID, rid, rcid, context string) {
	if output == nil {
		return "", "", "", ""
	}
	if len(output.Metadata) > 0 {
		chatID = output.Metadata[0]
	}
	if len(output.Metadata) > 1 {
		rid = output.Metadata[1]
	}
	if len(output.Metadata) > 2 {
		rcid = output.Metadata[2]
	}
	if rcid == "" {
		rcid = output.RCid
	}
	if len(output.Metadata) > 9 {
		context = output.Metadata[9]
	}
	return chatID, rid, rcid, context
}
