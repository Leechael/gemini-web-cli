package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
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
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/notebooks/view"))
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
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/notebooks/view"))
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
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/notebooks/view"))
	if err != nil {
		return err
	}
	if rejectCode != 0 {
		return fmt.Errorf("RemoveNotebookSource rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeRemoveNotebookSource(body)
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
