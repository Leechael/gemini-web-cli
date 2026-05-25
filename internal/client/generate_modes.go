package client

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

func calculateDelta(prev, current string) string {
	if prev == "" {
		return current
	}
	if strings.HasPrefix(current, prev) {
		return current[len(prev):]
	}
	return current
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateURLSafeToken(nbytes int) string {
	b := make([]byte, nbytes)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateHexUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *Client) resolveGenerationMode(prompt string, uploads []*UploadResult) string {
	mode := strings.ToLower(strings.TrimSpace(c.generationMode))
	switch mode {
	case "text", "", "auto":
	case "video", "image-to-video", "music":
		return mode
	default:
		return ""
	}
	if mode == "text" {
		return ""
	}
	lower := strings.ToLower(prompt)
	if len(uploads) > 0 && (strings.Contains(lower, "video") || strings.Contains(lower, "视频")) {
		return "image-to-video"
	}
	if strings.Contains(lower, "music") || strings.Contains(lower, "song") || strings.Contains(lower, "audio") || strings.Contains(lower, "音乐") || strings.Contains(lower, "歌曲") {
		return "music"
	}
	if strings.Contains(lower, "video") || strings.Contains(lower, "视频") {
		return "video"
	}
	return ""
}

func modelSelector(model *types.Model) int {
	if model == nil {
		return 1
	}
	header := model.Headers[types.ModelHeaderKey]
	if header == "" {
		return 1
	}
	var arr []any
	if err := json.Unmarshal([]byte(header), &arr); err != nil {
		return 1
	}
	if len(arr) > 14 {
		if f, ok := arr[14].(float64); ok && f != 0 {
			return int(f)
		}
	}
	return 1
}
