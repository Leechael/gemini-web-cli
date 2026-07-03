package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client/transport"
)

// CallStreamGenerate sends a StreamGenerate request and returns the response body.
func (c *Client) CallStreamGenerate(ctx context.Context, req transport.StreamGenerateRequest) (io.ReadCloser, error) {
	return c.callStreamGenerate(ctx, req, c.session())
}

func (c *Client) callStreamGenerate(ctx context.Context, req transport.StreamGenerateRequest, s sessionSnapshot) (io.ReadCloser, error) {
	const maxAttempts = 3

	req.Client = c.httpClient
	req.UserAgent = userAgent

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req.URL = transport.BuildStreamGenerateURL(transport.StreamURLConfig{
			BaseURL:     baseURL,
			AccountPath: c.accountPath,
			ReqID:       c.nextReqID(),
			Language:    s.language,
			BuildLabel:  s.buildLabel,
			SessionID:   s.sessionID,
		})

		body, err := transport.PostStreamGenerate(ctx, req)
		if err == nil {
			return body, nil
		}
		if statusErr, ok := err.(*transport.HTTPStatusError); ok && statusErr.StatusCode == 429 {
			return nil, &RateLimitError{StatusCode: statusErr.StatusCode}
		}
		lastErr = err
		if attempt == maxAttempts || !isRetryableStreamGenerateError(err) {
			break
		}
		fmt.Fprintf(logWriter, "stream request failed (attempt %d/%d), retrying: %v\n", attempt, maxAttempts, err)
		if err := sleepBeforeStreamRetry(ctx, attempt); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func isRetryableStreamGenerateError(err error) bool {
	var statusErr *transport.HTTPStatusError
	if errors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case 502, 503, 504:
			return true
		default:
			return false
		}
	}

	var requestErr *transport.StreamRequestError
	if !errors.As(err, &requestErr) {
		return false
	}
	if errors.Is(requestErr.Err, io.ErrUnexpectedEOF) || errors.Is(requestErr.Err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(requestErr.Err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(requestErr.Err.Error())
	return strings.Contains(msg, "unexpected eof") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "server closed idle connection")
}

func sleepBeforeStreamRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt*250) * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
