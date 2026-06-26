package client

import (
	"context"
	"io"

	"github.com/Leechael/gemini-web-cli/internal/client/transport"
)

// CallStreamGenerate sends a StreamGenerate request and returns the response body.
func (c *Client) CallStreamGenerate(ctx context.Context, req transport.StreamGenerateRequest) (io.ReadCloser, error) {
	return c.callStreamGenerate(ctx, req, c.session())
}

func (c *Client) callStreamGenerate(ctx context.Context, req transport.StreamGenerateRequest, s sessionSnapshot) (io.ReadCloser, error) {
	req.Client = c.httpClient
	req.URL = transport.BuildStreamGenerateURL(transport.StreamURLConfig{
		BaseURL:     baseURL,
		AccountPath: c.accountPath,
		ReqID:       c.nextReqID(),
		Language:    s.language,
		BuildLabel:  s.buildLabel,
		SessionID:   s.sessionID,
	})
	req.UserAgent = userAgent
	body, err := transport.PostStreamGenerate(ctx, req)
	if err != nil {
		if statusErr, ok := err.(*transport.HTTPStatusError); ok && statusErr.StatusCode == 429 {
			return nil, &RateLimitError{StatusCode: statusErr.StatusCode}
		}
		return nil, err
	}
	return body, nil
}
