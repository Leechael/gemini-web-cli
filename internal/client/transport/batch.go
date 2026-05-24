package transport

import (
	"fmt"
	"net/url"
	"strings"
)

// BatchURLConfig contains the query and path inputs for a batchexecute URL.
type BatchURLConfig struct {
	BaseURL     string
	AccountPath string
	RPCIDs      []string
	ReqID       int
	Language    string
	BuildLabel  string
	SessionID   string
	SourcePath  string
}

// BuildBatchURL constructs the Gemini batchexecute URL.
func BuildBatchURL(cfg BatchURLConfig) string {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://gemini.google.com"
	}

	language := cfg.Language
	if language == "" {
		language = "en"
	}

	sourcePath := cfg.SourcePath
	if sourcePath == "" {
		sourcePath = cfg.AccountPath + "/app"
	}

	params := url.Values{}
	params.Set("rpcids", strings.Join(cfg.RPCIDs, ","))
	params.Set("_reqid", fmt.Sprintf("%d", cfg.ReqID))
	params.Set("rt", "c")
	params.Set("hl", language)
	params.Set("pageId", "none")
	params.Set("source-path", sourcePath)
	if cfg.BuildLabel != "" {
		params.Set("bl", cfg.BuildLabel)
	}
	if cfg.SessionID != "" {
		params.Set("f.sid", cfg.SessionID)
	}

	return base + cfg.AccountPath + "/_/BardChatUi/data/batchexecute?" + params.Encode()
}
