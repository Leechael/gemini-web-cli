package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// accountClient is the subset of *client.Client the server uses. It exists
// so accountPool can be unit-tested with fakes.
type accountClient interface {
	Init(ctx context.Context) error
	Close()
	FetchAndCacheModels(ctx context.Context) error
	AvailableModels() []types.Model
	ResolveModel(name string) *types.Model

	ReadChat(ctx context.Context, cid string, maxTurns int) ([]types.ChatTurn, error)
	FetchLatestChatResponse(ctx context.Context, cid string) (*client.LatestResponse, error)
	SendMessage(ctx context.Context, prompt string, metadata []string, model *types.Model, notebook string) (*types.ModelOutput, error)
	SendMessageStream(ctx context.Context, prompt string, metadata []string, model *types.Model, notebook string, cb client.StreamCallback) (*types.ModelOutput, error)
	GenerateContent(ctx context.Context, prompt string, model *types.Model, notebook string) (*types.ModelOutput, error)
	GenerateContentStream(ctx context.Context, prompt string, model *types.Model, notebook string, cb client.StreamCallback) (*types.ModelOutput, error)

	CreateAndStartDeepResearch(ctx context.Context, prompt string, model *types.Model) (*types.DeepResearchPlan, error)
	CheckDeepResearch(ctx context.Context, cid string) (*client.ResearchStatus, error)
	GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error)
	SendMessageDeepResearch(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error)
	ListResearchReportsPage(ctx context.Context, count int, cursor string) ([]rpcs.ResearchReport, string, error)

	CreateNotebook(ctx context.Context, title string) (string, error)
	GetNotebook(ctx context.Context, notebookID string) (*rpcs.Notebook, error)
	ListNotebookChats(ctx context.Context, notebookID string) ([]types.ChatItem, error)
	UploadFile(ctx context.Context, filePath string) (*client.UploadResult, error)
	AddNotebookSource(ctx context.Context, notebookID string, fileName string, mimeType string, uploadToken string) (*rpcs.Notebook, error)
	AddNotebookURLSource(ctx context.Context, notebookID string, url string) (*rpcs.Notebook, error)
	RemoveNotebookSource(ctx context.Context, sourceResource string) error
}

// fatalStreamError wraps an error that must not trigger failover because the
// stream already emitted deltas to the caller.
type fatalStreamError struct{ err error }

func (e fatalStreamError) Error() string { return e.err.Error() }

// accountPool fans server requests out over multiple Gemini accounts.
//
// Routing rules:
//   - New conversations/notebooks/research tasks pick the next account in
//     round-robin order; on failure the remaining accounts are tried.
//   - Follow-up requests for an existing chat/research id or notebook id are
//     pinned to the account that owns the resource (recorded at creation, or
//     discovered by probing each account on first use).
type accountPool struct {
	clients []accountClient
	sources []string

	mu             sync.Mutex
	rr             int
	chatOwners     map[string]int
	notebookOwners map[string]int
}

func newAccountPool(clients []accountClient, sources []string) *accountPool {
	return &accountPool{
		clients:        clients,
		sources:        sources,
		chatOwners:     make(map[string]int),
		notebookOwners: make(map[string]int),
	}
}

// Len returns the number of accounts in the pool.
func (p *accountPool) Len() int { return len(p.clients) }

// Sources returns the cookie file path per account, for display.
func (p *accountPool) Sources() []string { return p.sources }

// Init initializes every account. It succeeds when at least one account
// initializes; per-account failures are logged.
func (p *accountPool) Init(ctx context.Context) error {
	var failures []string
	for i, c := range p.clients {
		if err := c.Init(ctx); err != nil {
			msg := fmt.Sprintf("%s: %s", p.accountLabel(i), sanitizeUpstreamError(err.Error()))
			failures = append(failures, msg)
			log.Printf("%s init failed: %s", p.accountLabel(i), sanitizeUpstreamError(err.Error()))
			continue
		}
		log.Printf("%s ready", p.accountLabel(i))
	}
	if len(failures) == len(p.clients) {
		return fmt.Errorf("all accounts failed to initialize:\n  %s", strings.Join(failures, "\n  "))
	}
	return nil
}

// FetchAndCacheModels caches the model list from the first account that
// answers; the catalog is shared across accounts.
func (p *accountPool) FetchAndCacheModels(ctx context.Context) error {
	var lastErr error
	for _, c := range p.clients {
		if err := c.FetchAndCacheModels(ctx); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

// Close releases every account.
func (p *accountPool) Close() {
	for _, c := range p.clients {
		c.Close()
	}
}

// AvailableModels returns the model list from the first account that has one.
func (p *accountPool) AvailableModels() []types.Model {
	for _, c := range p.clients {
		if models := c.AvailableModels(); len(models) > 0 {
			return models
		}
	}
	return nil
}

// ResolveModel resolves a model name against the first account that knows it.
func (p *accountPool) ResolveModel(name string) *types.Model {
	for _, c := range p.clients {
		if m := c.ResolveModel(name); m != nil {
			return m
		}
	}
	return nil
}

// accountLabel is the 1-based log identity for an account, e.g.
// "account 1/2 (alice.json)". Indexes in logs match how operators count
// accounts, not the 0-based slice position.
func (p *accountPool) accountLabel(idx int) string {
	n := len(p.clients)
	if idx < 0 || idx >= n {
		return fmt.Sprintf("account ?/%d", n)
	}
	var short string
	if idx < len(p.sources) {
		short = shortSourceName(p.sources[idx])
	}
	if short == "" {
		return fmt.Sprintf("account %d/%d", idx+1, n)
	}
	return fmt.Sprintf("account %d/%d (%s)", idx+1, n, short)
}

func shortSourceName(name string) string {
	if name == "" {
		return ""
	}
	path := name
	if i := strings.Index(name, " ("); i >= 0 {
		path = name[:i]
	}
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) {
		return path
	}
	return base
}

func (p *accountPool) labelForChat(cid string) string {
	if cid == "" {
		return ""
	}
	p.mu.Lock()
	idx, ok := p.chatOwners[cid]
	p.mu.Unlock()
	if !ok {
		return ""
	}
	return p.accountLabel(idx)
}

// forNew runs fn against accounts in round-robin order until one succeeds.
// It returns the winning account index so callers can record affinity.
// A fatalStreamError stops the failover immediately.
func (p *accountPool) forNew(fn func(c accountClient) error) (int, error) {
	p.mu.Lock()
	start := p.rr % len(p.clients)
	p.rr++
	p.mu.Unlock()

	var lastErr error
	for i := 0; i < len(p.clients); i++ {
		idx := (start + i) % len(p.clients)
		err := fn(p.clients[idx])
		if err == nil {
			return idx, nil
		}
		if fatal, ok := err.(fatalStreamError); ok {
			return -1, fatal.err
		}
		log.Printf("%s failed, trying next: %s", p.accountLabel(idx), sanitizeUpstreamError(err.Error()))
		lastErr = err
	}
	return -1, lastErr
}

func (p *accountPool) recordChat(cid string, idx int) {
	if cid == "" || idx < 0 {
		return
	}
	p.mu.Lock()
	p.chatOwners[cid] = idx
	p.mu.Unlock()
	log.Printf("%s chat_id=%s", p.accountLabel(idx), cid)
}

func (p *accountPool) recordChatOutput(output *types.ModelOutput, idx int) {
	if output == nil || len(output.Metadata) == 0 {
		return
	}
	p.recordChat(output.Metadata[0], idx)
}

// chatOwner resolves the account owning a chat or research id, probing every
// account on first use (e.g. after a restart wiped the in-memory affinity).
func (p *accountPool) chatOwner(ctx context.Context, cid string) (accountClient, error) {
	p.mu.Lock()
	idx, ok := p.chatOwners[cid]
	p.mu.Unlock()
	if ok {
		return p.clients[idx], nil
	}
	for i, c := range p.clients {
		turns, err := c.ReadChat(ctx, cid, 1)
		if err == nil && len(turns) > 0 {
			p.recordChat(cid, i)
			return c, nil
		}
	}
	return nil, fmt.Errorf("chat %q not found on any account", cid)
}

func normalizeNotebookID(id string) string {
	return strings.TrimPrefix(id, "notebooks/")
}

func (p *accountPool) recordNotebook(id string, idx int) {
	id = normalizeNotebookID(id)
	if id == "" || idx < 0 {
		return
	}
	p.mu.Lock()
	p.notebookOwners[id] = idx
	p.mu.Unlock()
	log.Printf("%s notebook=%s", p.accountLabel(idx), id)
}

// errNotebookNotFound marks a confirmed negative ownership probe: every
// account answered and none owns the notebook. Probe failures (timeouts,
// 429, expired sessions) return a plain error instead so handlers can surface
// them rather than answering 404.
var errNotebookNotFound = errors.New("notebook not found on any account")

// notebookOwner resolves the account owning a notebook, probing every account
// on first use.
func (p *accountPool) notebookOwner(ctx context.Context, id string) (accountClient, int, error) {
	key := normalizeNotebookID(id)
	p.mu.Lock()
	idx, ok := p.notebookOwners[key]
	p.mu.Unlock()
	if ok {
		return p.clients[idx], idx, nil
	}
	var probeErrs []string
	for i, c := range p.clients {
		nb, err := c.GetNotebook(ctx, id)
		if err == nil && nb != nil {
			p.recordNotebook(key, i)
			return c, i, nil
		}
		if err != nil {
			probeErrs = append(probeErrs, fmt.Sprintf("%s: %s", p.accountLabel(i), sanitizeUpstreamError(err.Error())))
		}
	}
	if len(probeErrs) > 0 {
		return nil, -1, fmt.Errorf("probing notebook %q ownership failed:\n  %s", id, strings.Join(probeErrs, "\n  "))
	}
	return nil, -1, fmt.Errorf("notebook %q: %w", id, errNotebookNotFound)
}

// FetchLatestChatResponse routes to the account owning the chat.
func (p *accountPool) FetchLatestChatResponse(ctx context.Context, cid string) (*client.LatestResponse, error) {
	c, err := p.chatOwner(ctx, cid)
	if err != nil {
		return nil, err
	}
	return c.FetchLatestChatResponse(ctx, cid)
}

// SendMessage continues an existing chat when metadata carries a chat id,
// otherwise starts a new chat on the next account with failover.
func (p *accountPool) SendMessage(ctx context.Context, prompt string, metadata []string, model *types.Model, notebook string) (*types.ModelOutput, error) {
	if len(metadata) > 0 && metadata[0] != "" {
		c, err := p.chatOwner(ctx, metadata[0])
		if err != nil {
			return nil, err
		}
		return c.SendMessage(ctx, prompt, metadata, model, notebook)
	}
	var output *types.ModelOutput
	idx, err := p.forNew(func(c accountClient) error {
		out, err := c.SendMessage(ctx, prompt, metadata, model, notebook)
		if err != nil {
			return err
		}
		output = out
		return nil
	})
	if err != nil {
		return nil, err
	}
	p.recordChatOutput(output, idx)
	return output, nil
}

// SendMessageStream mirrors SendMessage for streaming. Failover only happens
// when the failing account has not emitted any delta yet.
func (p *accountPool) SendMessageStream(ctx context.Context, prompt string, metadata []string, model *types.Model, notebook string, cb client.StreamCallback) (*types.ModelOutput, error) {
	if len(metadata) > 0 && metadata[0] != "" {
		c, err := p.chatOwner(ctx, metadata[0])
		if err != nil {
			return nil, err
		}
		return c.SendMessageStream(ctx, prompt, metadata, model, notebook, cb)
	}
	var output *types.ModelOutput
	idx, err := p.forNew(func(c accountClient) error {
		// Buffer metadata-only frames so a failing account cannot leak its
		// chat id to the caller before failover picks the winner.
		var buffered []*types.ModelOutput
		emitted := false
		out, err := c.SendMessageStream(ctx, prompt, metadata, model, notebook, func(o *types.ModelOutput) {
			if o.TextDelta != "" || o.ThoughtsDelta != "" {
				emitted = true
				for _, b := range buffered {
					cb(b)
				}
				buffered = nil
				cb(o)
				return
			}
			buffered = append(buffered, o)
		})
		if err != nil {
			if emitted {
				return fatalStreamError{err}
			}
			return err
		}
		// Successful stream that never produced a delta: deliver the held
		// metadata frames so the caller still learns the chat id.
		for _, b := range buffered {
			cb(b)
		}
		output = out
		return nil
	})
	if err != nil {
		return nil, err
	}
	p.recordChatOutput(output, idx)
	return output, nil
}

// GenerateContent starts a new chat on the next account with failover. When
// the chat is scoped to a notebook, it is created on the account owning the
// notebook — Gemini cannot attach a chat to another account's notebook.
func (p *accountPool) GenerateContent(ctx context.Context, prompt string, model *types.Model, notebook string) (*types.ModelOutput, error) {
	if notebook != "" {
		c, idx, err := p.notebookOwner(ctx, notebook)
		if err != nil {
			return nil, err
		}
		output, err := c.GenerateContent(ctx, prompt, model, notebook)
		if err != nil {
			return nil, err
		}
		p.recordChatOutput(output, idx)
		return output, nil
	}
	var output *types.ModelOutput
	idx, err := p.forNew(func(c accountClient) error {
		out, err := c.GenerateContent(ctx, prompt, model, notebook)
		if err != nil {
			return err
		}
		output = out
		return nil
	})
	if err != nil {
		return nil, err
	}
	p.recordChatOutput(output, idx)
	return output, nil
}

// GenerateContentStream mirrors GenerateContent for streaming. Failover only
// happens when the failing account has not emitted any delta yet.
func (p *accountPool) GenerateContentStream(ctx context.Context, prompt string, model *types.Model, notebook string, cb client.StreamCallback) (*types.ModelOutput, error) {
	if notebook != "" {
		c, idx, err := p.notebookOwner(ctx, notebook)
		if err != nil {
			return nil, err
		}
		output, err := c.GenerateContentStream(ctx, prompt, model, notebook, cb)
		if err != nil {
			return nil, err
		}
		p.recordChatOutput(output, idx)
		return output, nil
	}
	var output *types.ModelOutput
	idx, err := p.forNew(func(c accountClient) error {
		// Buffer metadata-only frames so a failing account cannot leak its
		// chat id to the caller before failover picks the winner.
		var buffered []*types.ModelOutput
		emitted := false
		out, err := c.GenerateContentStream(ctx, prompt, model, notebook, func(o *types.ModelOutput) {
			if o.TextDelta != "" || o.ThoughtsDelta != "" {
				emitted = true
				for _, b := range buffered {
					cb(b)
				}
				buffered = nil
				cb(o)
				return
			}
			buffered = append(buffered, o)
		})
		if err != nil {
			if emitted {
				return fatalStreamError{err}
			}
			return err
		}
		// Successful stream that never produced a delta: deliver the held
		// metadata frames so the caller still learns the chat id.
		for _, b := range buffered {
			cb(b)
		}
		output = out
		return nil
	})
	if err != nil {
		return nil, err
	}
	p.recordChatOutput(output, idx)
	return output, nil
}

// CreateAndStartDeepResearch starts a research task on the next account with
// failover and records the task's chat id for follow-up polling.
func (p *accountPool) CreateAndStartDeepResearch(ctx context.Context, prompt string, model *types.Model) (*types.DeepResearchPlan, error) {
	var plan *types.DeepResearchPlan
	idx, err := p.forNew(func(c accountClient) error {
		out, err := c.CreateAndStartDeepResearch(ctx, prompt, model)
		if err != nil {
			return err
		}
		plan = out
		return nil
	})
	if err != nil {
		return nil, err
	}
	if plan != nil {
		p.recordChat(plan.Cid, idx)
	}
	return plan, nil
}

// CheckDeepResearch routes to the account owning the research chat.
func (p *accountPool) CheckDeepResearch(ctx context.Context, cid string) (*client.ResearchStatus, error) {
	c, err := p.chatOwner(ctx, cid)
	if err != nil {
		return nil, err
	}
	return c.CheckDeepResearch(ctx, cid)
}

// GetDeepResearchResult routes to the account owning the research chat.
func (p *accountPool) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	c, err := p.chatOwner(ctx, cid)
	if err != nil {
		return "", nil, err
	}
	return c.GetDeepResearchResult(ctx, cid)
}

// SendMessageDeepResearch routes to the account owning the research chat.
func (p *accountPool) SendMessageDeepResearch(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	if len(metadata) == 0 || metadata[0] == "" {
		return nil, fmt.Errorf("research chat id is required in metadata")
	}
	c, err := p.chatOwner(ctx, metadata[0])
	if err != nil {
		return nil, err
	}
	return c.SendMessageDeepResearch(ctx, prompt, metadata, model)
}

// ListResearchReportsPage merges reports from every account, newest first.
// Cross-account pagination is not supported: the returned cursor is empty
// when the pool has more than one account.
func (p *accountPool) ListResearchReportsPage(ctx context.Context, count int, cursor string) ([]rpcs.ResearchReport, string, error) {
	if len(p.clients) == 1 {
		return p.clients[0].ListResearchReportsPage(ctx, count, cursor)
	}
	var all []rpcs.ResearchReport
	var failures []string
	for i, c := range p.clients {
		reports, _, err := c.ListResearchReportsPage(ctx, count, "")
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", p.accountLabel(i), sanitizeUpstreamError(err.Error())))
			continue
		}
		all = append(all, reports...)
	}
	if len(all) == 0 && len(failures) > 0 {
		return nil, "", fmt.Errorf("listing research reports failed:\n  %s", strings.Join(failures, "\n  "))
	}
	for _, f := range failures {
		log.Printf("research list partial failure: %s", f)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt > all[j].CreatedAt })
	if len(all) > count {
		all = all[:count]
	}
	return all, "", nil
}

// CreateNotebook creates a notebook on the next account with failover and
// records its owner.
func (p *accountPool) CreateNotebook(ctx context.Context, title string) (string, error) {
	var resource string
	idx, err := p.forNew(func(c accountClient) error {
		r, err := c.CreateNotebook(ctx, title)
		if err != nil {
			return err
		}
		resource = r
		return nil
	})
	if err != nil {
		return "", err
	}
	p.recordNotebook(resource, idx)
	return resource, nil
}

// GetNotebook routes to the account owning the notebook. It returns
// (nil, nil) when probes confirmed no account owns the id, so handlers can
// answer 404; probe failures are returned as errors instead.
func (p *accountPool) GetNotebook(ctx context.Context, id string) (*rpcs.Notebook, error) {
	c, _, err := p.notebookOwner(ctx, id)
	if err != nil {
		if errors.Is(err, errNotebookNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return c.GetNotebook(ctx, id)
}

// ListNotebookChats routes to the account owning the notebook.
func (p *accountPool) ListNotebookChats(ctx context.Context, id string) ([]types.ChatItem, error) {
	c, _, err := p.notebookOwner(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.ListNotebookChats(ctx, id)
}

// AddNotebookURLSource routes to the account owning the notebook.
func (p *accountPool) AddNotebookURLSource(ctx context.Context, id string, url string) (*rpcs.Notebook, error) {
	c, _, err := p.notebookOwner(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.AddNotebookURLSource(ctx, id, url)
}

// AddNotebookFileSource uploads a file and attaches it to the notebook on the
// account owning the notebook. Upload and attach must share one account
// because upload tokens are account-bound.
func (p *accountPool) AddNotebookFileSource(ctx context.Context, id string, path string) (*rpcs.Notebook, error) {
	c, _, err := p.notebookOwner(ctx, id)
	if err != nil {
		return nil, err
	}
	u, err := c.UploadFile(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	return c.AddNotebookSource(ctx, id, u.FileName, u.MimeType, u.ID)
}

// RemoveNotebookSource routes to the account owning the notebook that the
// source resource belongs to ("notebooks/<id>/sources/<sid>").
func (p *accountPool) RemoveNotebookSource(ctx context.Context, resource string) error {
	notebookID := resource
	if idx := strings.Index(resource, "/sources/"); idx >= 0 {
		notebookID = resource[:idx]
	}
	c, _, err := p.notebookOwner(ctx, notebookID)
	if err != nil {
		return err
	}
	return c.RemoveNotebookSource(ctx, resource)
}
