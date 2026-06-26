package client

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

var baseURL = "https://gemini.google.com"

const (
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	cookieDomain = ".google.com"
)

// RateLimitError is returned when the server responds with HTTP 429.
type RateLimitError struct {
	StatusCode int
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited by server (HTTP %d)", e.StatusCode)
}

// Client communicates with the Gemini web API.
type Client struct {
	httpClient   *http.Client
	reqID        atomic.Int64
	accountIndex *int
	accountPath  string // "" or "/u/N"
	model        *types.Model
	proxy        string
	verbose      bool
	timeout      time.Duration

	// sessionMu protects fields refreshed by Init().
	sessionMu   sync.RWMutex
	accessToken string
	buildLabel  string
	sessionID   string
	language    string // extracted from init page, default "en"
	pushID      string // extracted from init page, default "feeds/mcudyrk2a4khkz"

	// generationMode is per-request in server mode; CLI sets it before each call.
	generationMu   sync.RWMutex
	generationMode string

	// Cookies for persistence tracking
	cookieMu     sync.RWMutex
	ExtraCookies map[string]string

	// Dynamic model discovery
	AccountStatus types.AccountStatus
	modelRegistry map[string]*types.Model // dynamic model registry
}

// Config holds client construction parameters.
type Config struct {
	Secure1PSID   string
	Secure1PSIDTS string
	ExtraCookies  map[string]string
	Proxy         string
	AccountIndex  *int
	Model         *types.Model
	Verbose       bool
	Timeout       time.Duration
}

// New creates a new Client (not yet initialized).
func New(cfg Config) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(baseURL)

	// Set required cookies
	cookies := []*http.Cookie{
		{Name: "__Secure-1PSID", Value: cfg.Secure1PSID, Domain: cookieDomain, Path: "/", Secure: true, HttpOnly: true},
	}
	if cfg.Secure1PSIDTS != "" {
		cookies = append(cookies, &http.Cookie{Name: "__Secure-1PSIDTS", Value: cfg.Secure1PSIDTS, Domain: cookieDomain, Path: "/", Secure: true, HttpOnly: true})
	}
	for k, v := range cfg.ExtraCookies {
		cookies = append(cookies, &http.Cookie{Name: k, Value: v, Domain: cookieDomain, Path: "/"})
	}
	jar.SetCookies(u, cookies)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ForceAttemptHTTP2 = true
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	// Generate random reqID
	n, _ := rand.Int(rand.Reader, big.NewInt(90000))

	// Account path for multi-account support
	var accountPath string
	if cfg.AccountIndex != nil {
		accountPath = fmt.Sprintf("/u/%d", *cfg.AccountIndex)
	}

	c := &Client{
		httpClient: &http.Client{
			Jar:       jar,
			Transport: transport,
			Timeout:   timeout,
		},
		accountIndex: cfg.AccountIndex,
		accountPath:  accountPath,
		model:        cfg.Model,
		proxy:        cfg.Proxy,
		verbose:      cfg.Verbose,
		timeout:      timeout,
		ExtraCookies: cfg.ExtraCookies,
	}
	c.reqID.Store(n.Int64() + 10000)
	return c, nil
}

// Init fetches the access token and session data from the Gemini app page.
func (c *Client) Init(ctx context.Context) error {
	appURL := c.appPath()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+appURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("init request failed: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if strings.Contains(finalURL, "accounts.google.com") {
		return fmt.Errorf("session expired — redirected to Google login. Re-import cookies with: gemini-web-cli import '<cookie_string>'")
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 429 {
			return &RateLimitError{StatusCode: resp.StatusCode}
		}
		return fmt.Errorf("init returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading init response: %w", err)
	}

	htmlBody := string(body)

	token := extractRegex(htmlBody, `"SNlM0e"\s*:\s*"([^"]*)"`)
	if token == "" {
		if strings.Contains(htmlBody, "accounts.google.com/ServiceLogin") || strings.Contains(htmlBody, "accounts.google.com/v3/signin") {
			return fmt.Errorf("session expired — page redirected to Google login. Re-import cookies with: gemini-web-cli import '<cookie_string>'")
		}
		if len(htmlBody) < 1000 {
			return fmt.Errorf("failed to extract access token — unexpected page content (got %d bytes). Cookies may be invalid or expired. Re-import with: gemini-web-cli import '<cookie_string>'", len(htmlBody))
		}
		return fmt.Errorf("failed to extract access token (SNlM0e) — cookies may be expired. Re-import with: gemini-web-cli import '<cookie_string>'")
	}

	bl := extractRegex(htmlBody, `"cfb2h"\s*:\s*"([^"]*)"`)
	sid := extractRegex(htmlBody, `"FdrFJe"\s*:\s*"([^"]*)"`)
	lang := extractRegex(htmlBody, `"TuX5cc"\s*:\s*"([^"]*)"`)
	if lang == "" {
		lang = "en"
	}
	pid := extractRegex(htmlBody, `"qKIAYe"\s*:\s*"([^"]*)"`)
	if pid == "" {
		pid = "feeds/mcudyrk2a4khkz"
	}

	c.sessionMu.Lock()
	c.accessToken = token
	c.buildLabel = bl
	c.sessionID = sid
	c.language = lang
	c.pushID = pid
	c.sessionMu.Unlock()

	if c.verbose {
		fmt.Fprintf(logWriter, "Init OK: token=%s... bl=%s sid=%s lang=%s push=%s\n", token[:min(8, len(token))], bl, sid, lang, pid)
	}

	return nil
}

// SetGenerationMode selects a browser generation mode for the next request.
func (c *Client) SetGenerationMode(mode string) {
	c.generationMu.Lock()
	defer c.generationMu.Unlock()
	c.generationMode = mode
}

func (c *Client) generationModeSnapshot() string {
	c.generationMu.RLock()
	defer c.generationMu.RUnlock()
	return c.generationMode
}

// sessionSnapshot holds a consistent copy of session fields for a single request.
type sessionSnapshot struct {
	accessToken string
	buildLabel  string
	sessionID   string
	language    string
	pushID      string
}

// session returns a consistent snapshot of the session fields under read lock.
func (c *Client) session() sessionSnapshot {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return sessionSnapshot{
		accessToken: c.accessToken,
		buildLabel:  c.buildLabel,
		sessionID:   c.sessionID,
		language:    c.language,
		pushID:      c.pushID,
	}
}

// Close releases resources.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// appPath returns the path for this account: "/app" or "/u/N/app".
func (c *Client) appPath() string {
	return c.accountPath + "/app"
}

func (c *Client) nextReqID() int {
	return int(c.reqID.Add(100000) - 100000)
}

func (c *Client) commonHeaders() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	h.Set("Host", "gemini.google.com")
	h.Set("Origin", baseURL)
	h.Set("Referer", baseURL+"/")
	h.Set("User-Agent", userAgent)
	h.Set("X-Same-Domain", "1")
	return h
}

// streamURL constructs the StreamGenerate URL with account path prefix.
func (c *Client) streamURL() string {
	params := url.Values{}
	params.Set("_reqid", fmt.Sprintf("%d", c.nextReqID()))
	params.Set("rt", "c")
	params.Set("hl", c.language)
	params.Set("pageId", "none")
	if c.buildLabel != "" {
		params.Set("bl", c.buildLabel)
	}
	if c.sessionID != "" {
		params.Set("f.sid", c.sessionID)
	}

	// Account path prefix goes into the URL path, not just params
	path := c.accountPath + "/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate"
	return baseURL + path + "?" + params.Encode()
}

// batchURL constructs the batchexecute URL with account path prefix.
func (c *Client) batchURL(rpcIDs []string, sourcePath string) string {
	params := url.Values{}
	params.Set("rpcids", strings.Join(rpcIDs, ","))
	params.Set("_reqid", fmt.Sprintf("%d", c.nextReqID()))
	params.Set("rt", "c")
	params.Set("hl", c.language)
	params.Set("pageId", "none")
	if sourcePath != "" {
		params.Set("source-path", sourcePath)
	} else {
		params.Set("source-path", c.appPath())
	}
	if c.buildLabel != "" {
		params.Set("bl", c.buildLabel)
	}
	if c.sessionID != "" {
		params.Set("f.sid", c.sessionID)
	}

	// Account path prefix goes into the URL path
	return baseURL + c.accountPath + "/_/BardChatUi/data/batchexecute?" + params.Encode()
}

// logWriter is stderr for debug output.
var logWriter io.Writer = io.Discard

// SetVerbose enables debug logging to stderr.
func SetVerbose(w io.Writer) {
	logWriter = w
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
