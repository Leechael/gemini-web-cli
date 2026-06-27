package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type chatPrefixHash struct {
	Hash         string
	ParentHash   string
	MessageCount uint32
	Completed    bool
}

func chatHistoryHashes(messages []chatMessage) ([]chatPrefixHash, error) {
	prefixes := make([]chatPrefixHash, 0, len(messages))
	parent := ""
	for i, msg := range messages {
		role, content, err := canonicalChatMessage(msg)
		if err != nil {
			return nil, err
		}
		messageHash := hashParts("msg:v1", role, content)
		nodeHash := hashParts("node:v1", parent, messageHash)
		prefixes = append(prefixes, chatPrefixHash{
			Hash:         nodeHash,
			ParentHash:   parent,
			MessageCount: uint32(i + 1),
			Completed:    role == "assistant",
		})
		parent = nodeHash
	}
	return prefixes, nil
}

func completedChatPrefixes(messages []chatMessage) ([]chatPrefixHash, error) {
	prefixes, err := chatHistoryHashes(messages)
	if err != nil {
		return nil, err
	}
	completed := make([]chatPrefixHash, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix.Completed {
			completed = append(completed, prefix)
		}
	}
	return completed, nil
}

func completedChatRoot(messages []chatMessage) (chatPrefixHash, bool, error) {
	prefixes, err := chatHistoryHashes(messages)
	if err != nil {
		return chatPrefixHash{}, false, err
	}
	if len(prefixes) == 0 {
		return chatPrefixHash{}, false, nil
	}
	last := prefixes[len(prefixes)-1]
	if !last.Completed {
		return chatPrefixHash{}, false, nil
	}
	return last, true, nil
}

func canonicalChatMessage(msg chatMessage) (role string, content string, err error) {
	role = strings.ToLower(strings.TrimSpace(msg.Role))
	switch role {
	case "system", "user", "assistant":
		return role, msg.Content, nil
	default:
		return "", "", fmt.Errorf("unsupported message role %q", msg.Role)
	}
}

func hashParts(parts ...string) string {
	h := sha256.New()
	for i, part := range parts {
		if i > 0 {
			h.Write([]byte{0})
		}
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func flattenChatMessages(messages []chatMessage) (string, error) {
	var parts []string
	for _, msg := range messages {
		role, content, err := canonicalChatMessage(msg)
		if err != nil {
			return "", err
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%s]\n%s", roleLabel(role), content))
	}
	return strings.Join(parts, "\n\n"), nil
}

func roleLabel(role string) string {
	switch role {
	case "system":
		return "System"
	case "assistant":
		return "Assistant"
	default:
		return "User"
	}
}
