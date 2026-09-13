package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// CreateNotebook creates a notebook with the given title and returns its
// resource name ("notebooks/<uuid>").
func (c *Client) CreateNotebook(ctx context.Context, title string) (string, error) {
	rpcID, payload := rpcs.EncodeCreateNotebook(title)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/notebooks/create"))
	if err != nil {
		return "", err
	}
	if rejectCode != 0 {
		return "", fmt.Errorf("CreateNotebook rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeCreateNotebook(body)
}

// AddNotebookSource attaches an already-uploaded file to a notebook as a
// source and returns the refreshed notebook. The upload token is the
// "/contrib_service/..." string returned by UploadFile.
func (c *Client) AddNotebookSource(ctx context.Context, notebookID string, fileName string, mimeType string, uploadToken string) (*rpcs.Notebook, error) {
	rpcID, payload := rpcs.EncodeAddNotebookSource(notebookID, fileName, mimeType, uploadToken)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath(notebookPagePath(notebookID)))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("AddNotebookSource rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeAddNotebookSource(body)
}

// AddNotebookURLSource attaches a web URL to a notebook as a source and
// returns the refreshed notebook.
func (c *Client) AddNotebookURLSource(ctx context.Context, notebookID string, url string) (*rpcs.Notebook, error) {
	rpcID, payload := rpcs.EncodeAddNotebookURLSource(notebookID, url)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath(notebookPagePath(notebookID)))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("AddNotebookURLSource rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeAddNotebookSource(body)
}

// RemoveNotebookSource removes a source from a notebook. The source must be a
// full resource name ("notebooks/<uuid>/sources/<sid>").
func (c *Client) RemoveNotebookSource(ctx context.Context, sourceResource string) error {
	rpcID, payload := rpcs.EncodeRemoveNotebookSource(sourceResource)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath(notebookPagePath(sourceResource)))
	if err != nil {
		return err
	}
	if rejectCode != 0 {
		return fmt.Errorf("RemoveNotebookSource rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeRemoveNotebookSource(body)
}

// ListNotebookChats returns the first page of chats belonging to a notebook
// (page size 10), newest first. Later pages are not requested; a notebook
// with more than 10 chats is truncated. The id may be passed with or without
// the "notebooks/" prefix.
func (c *Client) ListNotebookChats(ctx context.Context, notebookID string) ([]types.ChatItem, error) {
	resource := notebookID
	if !strings.HasPrefix(resource, "notebooks/") {
		resource = "notebooks/" + resource
	}
	rpcID, payload := rpcs.EncodeListChatsRaw(rpcs.ListChatsPayload{PageSize: 10, NotebookResource: resource})
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/notebooks/view"))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("ListNotebookChats rejected with code=%d", rejectCode)
	}
	items, _, err := rpcs.DecodeListChats(body)
	if err != nil {
		return nil, err
	}
	out := make([]types.ChatItem, 0, len(items))
	for _, it := range items {
		chatItem := types.ChatItem{Cid: it.Cid, Title: it.Title}
		if it.UpdatedAtUnix != 0 {
			chatItem.UpdatedAt = time.Unix(it.UpdatedAtUnix, 0).UTC().Format("2006-01-02T15:04")
		}
		out = append(out, chatItem)
	}
	return out, nil
}

// GetNotebook fetches a notebook's metadata and source list. The id may be
// passed with or without the "notebooks/" prefix.
func (c *Client) GetNotebook(ctx context.Context, notebookID string) (*rpcs.Notebook, error) {
	rpcID, payload := rpcs.EncodeGetNotebook(notebookID, c.session().language)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/notebooks/view"))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetNotebook rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetNotebook(body)
}
