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

// CallStreamGenerate sends a StreamGenerate request and retries transient
// failures before an HTTP response is returned.
func (c *Client) CallStreamGenerate(ctx context.Context, req transport.StreamGenerateRequest) (io.ReadCloser, error) {
	return c.callStreamGenerateWithRetry(ctx, req, c.session())
}

func (c *Client) callStreamGenerateWithRetry(ctx context.Context, req transport.StreamGenerateRequest, s sessionSnapshot) (io.ReadCloser, error) {
	var body io.ReadCloser
	err := runStreamGenerateAttempts(ctx, func() (bool, error) {
		attemptBody, attemptErr := c.callStreamGenerate(ctx, req, s)
		if attemptErr != nil {
			body = nil
			return isRetryableStreamGenerateError(attemptErr), attemptErr
		}
		body = attemptBody
		return false, nil
	}, func(attempt, maxAttempts int, err error) {
		fmt.Fprintf(logWriter, "stream request failed (attempt %d/%d), retrying: %v\n", attempt, maxAttempts, err)
	})
	return body, err
}

const maxStreamGenerateAttempts = 3

func runStreamGenerateAttempts(
	ctx context.Context,
	attemptFn func() (bool, error),
	logRetry func(attempt, maxAttempts int, err error),
) error {
	var lastErr error
	for attempt := 1; attempt <= maxStreamGenerateAttempts; attempt++ {
		retry, err := attemptFn()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == maxStreamGenerateAttempts || !retry {
			return err
		}
		logRetry(attempt, maxStreamGenerateAttempts, err)
		if err := sleepBeforeStreamRetry(ctx, attempt); err != nil {
			return err
		}
	}
	return lastErr
}

func (c *Client) callStreamGenerate(ctx context.Context, req transport.StreamGenerateRequest, s sessionSnapshot) (io.ReadCloser, error) {
	req.Client = c.httpClient
	req.UserAgent = userAgent
	req.URL = transport.BuildStreamGenerateURL(transport.StreamURLConfig{
		BaseURL:     baseURL,
		AccountPath: c.accountPath,
		ReqID:       c.nextReqID(),
		Language:    s.language,
		BuildLabel:  s.buildLabel,
		SessionID:   s.sessionID,
	})

	body, err := transport.PostStreamGenerate(ctx, req)
	if statusErr, ok := err.(*transport.HTTPStatusError); ok && statusErr.StatusCode == 429 {
		return nil, &RateLimitError{StatusCode: statusErr.StatusCode}
	}
	return body, err
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
