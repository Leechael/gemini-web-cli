package server

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// fakeAccount implements accountClient with overridable hooks.
type fakeAccount struct {
	initFn            func(ctx context.Context) error
	generateContentFn func(ctx context.Context, prompt string) (*types.ModelOutput, error)
	genStreamFn       func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error)
	sendMessageFn     func(ctx context.Context, metadata []string) (*types.ModelOutput, error)
	readChatFn        func(ctx context.Context, cid string) ([]types.ChatTurn, error)
	getNotebookFn     func(ctx context.Context, id string) (*rpcs.Notebook, error)
	createNotebookFn  func(ctx context.Context, title string) (string, error)
	addURLSourceFn    func(ctx context.Context, id string, url string) (*rpcs.Notebook, error)
	listReportsFn     func(ctx context.Context) ([]rpcs.ResearchReport, string, error)
}

func (f *fakeAccount) Init(ctx context.Context) error {
	if f.initFn != nil {
		return f.initFn(ctx)
	}
	return nil
}
func (f *fakeAccount) Close()                                        {}
func (f *fakeAccount) FetchAndCacheModels(ctx context.Context) error { return nil }
func (f *fakeAccount) AvailableModels() []types.Model                { return nil }
func (f *fakeAccount) ResolveModel(name string) *types.Model         { return nil }

func (f *fakeAccount) ReadChat(ctx context.Context, cid string, maxTurns int) ([]types.ChatTurn, error) {
	if f.readChatFn != nil {
		return f.readChatFn(ctx, cid)
	}
	return nil, fmt.Errorf("read_chat rejected with code=1")
}

func (f *fakeAccount) FetchLatestChatResponse(ctx context.Context, cid string) (*client.LatestResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) SendMessage(ctx context.Context, prompt string, metadata []string, model *types.Model, notebook string) (*types.ModelOutput, error) {
	if f.sendMessageFn != nil {
		return f.sendMessageFn(ctx, metadata)
	}
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) SendMessageStream(ctx context.Context, prompt string, metadata []string, model *types.Model, notebook string, cb client.StreamCallback) (*types.ModelOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) GenerateContent(ctx context.Context, prompt string, model *types.Model, notebook string) (*types.ModelOutput, error) {
	if f.generateContentFn != nil {
		return f.generateContentFn(ctx, prompt)
	}
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) GenerateContentStream(ctx context.Context, prompt string, model *types.Model, notebook string, cb client.StreamCallback) (*types.ModelOutput, error) {
	if f.genStreamFn != nil {
		return f.genStreamFn(ctx, cb)
	}
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) CreateAndStartDeepResearch(ctx context.Context, prompt string, model *types.Model) (*types.DeepResearchPlan, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) CheckDeepResearch(ctx context.Context, cid string) (*client.ResearchStatus, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	return "", nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) SendMessageDeepResearch(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) ListResearchReportsPage(ctx context.Context, count int, cursor string) ([]rpcs.ResearchReport, string, error) {
	if f.listReportsFn != nil {
		return f.listReportsFn(ctx)
	}
	return nil, "", fmt.Errorf("not implemented")
}

func (f *fakeAccount) CreateNotebook(ctx context.Context, title string) (string, error) {
	if f.createNotebookFn != nil {
		return f.createNotebookFn(ctx, title)
	}
	return "", fmt.Errorf("not implemented")
}

func (f *fakeAccount) GetNotebook(ctx context.Context, notebookID string) (*rpcs.Notebook, error) {
	if f.getNotebookFn != nil {
		return f.getNotebookFn(ctx, notebookID)
	}
	return nil, nil
}

func (f *fakeAccount) ListNotebookChats(ctx context.Context, notebookID string) ([]types.ChatItem, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) UploadFile(ctx context.Context, filePath string) (*client.UploadResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) AddNotebookSource(ctx context.Context, notebookID string, fileName string, mimeType string, uploadToken string) (*rpcs.Notebook, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) AddNotebookURLSource(ctx context.Context, notebookID string, url string) (*rpcs.Notebook, error) {
	if f.addURLSourceFn != nil {
		return f.addURLSourceFn(ctx, notebookID, url)
	}
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeAccount) RemoveNotebookSource(ctx context.Context, sourceResource string) error {
	return fmt.Errorf("not implemented")
}

func TestPoolRoundRobinNewChats(t *testing.T) {
	var calls []int
	mk := func(idx int) *fakeAccount {
		return &fakeAccount{
			generateContentFn: func(ctx context.Context, prompt string) (*types.ModelOutput, error) {
				calls = append(calls, idx)
				return &types.ModelOutput{Text: "ok", Metadata: []string{fmt.Sprintf("c_%d", idx)}}, nil
			},
		}
	}
	pool := newAccountPool([]accountClient{mk(0), mk(1)}, nil)

	for i := 0; i < 4; i++ {
		if _, err := pool.GenerateContent(context.Background(), "hi", nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	want := []int{0, 1, 0, 1}
	if fmt.Sprint(calls) != fmt.Sprint(want) {
		t.Fatalf("rotation = %v, want %v", calls, want)
	}
}

func TestPoolFailoverNewChat(t *testing.T) {
	broken := &fakeAccount{
		generateContentFn: func(ctx context.Context, prompt string) (*types.ModelOutput, error) {
			return nil, fmt.Errorf("rate limited by server (HTTP 429)")
		},
	}
	healthy := &fakeAccount{
		generateContentFn: func(ctx context.Context, prompt string) (*types.ModelOutput, error) {
			return &types.ModelOutput{Text: "ok", Metadata: []string{"c_healthy"}}, nil
		},
	}
	pool := newAccountPool([]accountClient{broken, healthy}, nil)

	out, err := pool.GenerateContent(context.Background(), "hi", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "ok" {
		t.Fatalf("output = %q", out.Text)
	}
	// The follow-up must be pinned to the healthy account that owns the chat.
	metadata := []string{"c_healthy"}
	routed := false
	healthy.sendMessageFn = func(ctx context.Context, md []string) (*types.ModelOutput, error) {
		routed = true
		return &types.ModelOutput{Text: "continued"}, nil
	}
	broken.sendMessageFn = func(ctx context.Context, md []string) (*types.ModelOutput, error) {
		t.Fatal("follow-up routed to the wrong account")
		return nil, nil
	}
	if _, err := pool.SendMessage(context.Background(), "next", metadata, nil, ""); err != nil {
		t.Fatal(err)
	}
	if !routed {
		t.Fatal("follow-up did not reach the owning account")
	}
}

func TestPoolAllAccountsFailReturnsError(t *testing.T) {
	broken := &fakeAccount{
		generateContentFn: func(ctx context.Context, prompt string) (*types.ModelOutput, error) {
			return nil, fmt.Errorf("boom")
		},
	}
	pool := newAccountPool([]accountClient{broken, broken}, nil)
	if _, err := pool.GenerateContent(context.Background(), "hi", nil, ""); err == nil {
		t.Fatal("expected error when every account fails")
	}
}

func TestPoolStreamFailoverOnlyBeforeFirstDelta(t *testing.T) {
	// Account 0 emits a delta and then fails: no failover allowed, because the
	// caller already received bytes.
	emittedThenBroken := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			cb(&types.ModelOutput{TextDelta: "partial"})
			return nil, fmt.Errorf("stream died mid-flight")
		},
	}
	healthy := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			return &types.ModelOutput{Text: "ok", Metadata: []string{"c_1"}}, nil
		},
	}
	pool := newAccountPool([]accountClient{emittedThenBroken, healthy}, nil)

	var deltas int
	_, err := pool.GenerateContentStream(context.Background(), "hi", nil, "", func(o *types.ModelOutput) {
		deltas++
	})
	if err == nil || err.Error() != "stream died mid-flight" {
		t.Fatalf("err = %v, want mid-flight error surfaced without retry", err)
	}
	if deltas != 1 {
		t.Fatalf("deltas = %d, want 1", deltas)
	}

	// Account 1 fails before emitting anything: failover to account 2.
	brokenEarly := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			return nil, fmt.Errorf("rate limited by server (HTTP 429)")
		},
	}
	pool2 := newAccountPool([]accountClient{brokenEarly, healthy}, nil)
	out, err := pool2.GenerateContentStream(context.Background(), "hi", nil, "", func(o *types.ModelOutput) {})
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "ok" {
		t.Fatalf("output = %q", out.Text)
	}
}

func TestPoolChatOwnerProbeAfterRestart(t *testing.T) {
	// No affinity recorded (simulates a restart): the pool probes accounts
	// with ReadChat and pins the follow-up to the account that has the chat.
	stranger := &fakeAccount{} // ReadChat default: rejected
	owner := &fakeAccount{
		readChatFn: func(ctx context.Context, cid string) ([]types.ChatTurn, error) {
			if cid != "c_known" {
				return nil, fmt.Errorf("read_chat rejected with code=1")
			}
			return []types.ChatTurn{{UserPrompt: "hi"}}, nil
		},
	}
	routed := false
	owner.sendMessageFn = func(ctx context.Context, md []string) (*types.ModelOutput, error) {
		routed = true
		return &types.ModelOutput{Text: "continued"}, nil
	}
	pool := newAccountPool([]accountClient{stranger, owner}, nil)

	metadata := []string{"c_known"}
	if _, err := pool.SendMessage(context.Background(), "next", metadata, nil, ""); err != nil {
		t.Fatal(err)
	}
	if !routed {
		t.Fatal("follow-up did not reach the probed owner")
	}

	// A chat nobody owns must error, not silently start a new conversation.
	if _, err := pool.SendMessage(context.Background(), "next", []string{"c_unknown"}, nil, ""); err == nil {
		t.Fatal("expected error for a chat no account owns")
	}
}

func TestPoolNotebookAffinity(t *testing.T) {
	owner := &fakeAccount{
		createNotebookFn: func(ctx context.Context, title string) (string, error) {
			return "notebooks/nb-1", nil
		},
	}
	other := &fakeAccount{}
	added := false
	owner.addURLSourceFn = func(ctx context.Context, id string, url string) (*rpcs.Notebook, error) {
		added = true
		return &rpcs.Notebook{ResourceName: id}, nil
	}
	other.addURLSourceFn = func(ctx context.Context, id string, url string) (*rpcs.Notebook, error) {
		t.Fatal("add source routed to the wrong account")
		return nil, nil
	}
	pool := newAccountPool([]accountClient{owner, other}, nil)

	resource, err := pool.CreateNotebook(context.Background(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if resource != "notebooks/nb-1" {
		t.Fatalf("resource = %q", resource)
	}
	if _, err := pool.AddNotebookURLSource(context.Background(), "nb-1", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("add source did not reach the owning account")
	}
}

func TestPoolNotebookProbeAndGetMissing(t *testing.T) {
	owner := &fakeAccount{
		getNotebookFn: func(ctx context.Context, id string) (*rpcs.Notebook, error) {
			return &rpcs.Notebook{ResourceName: "notebooks/" + id}, nil
		},
	}
	pool := newAccountPool([]accountClient{&fakeAccount{}, owner}, nil)

	nb, err := pool.GetNotebook(context.Background(), "nb-1")
	if err != nil {
		t.Fatal(err)
	}
	if nb == nil || nb.ResourceName != "notebooks/nb-1" {
		t.Fatalf("notebook = %+v", nb)
	}

	// No account owns it: (nil, nil) so handlers answer 404.
	pool2 := newAccountPool([]accountClient{&fakeAccount{}}, nil)
	nb, err = pool2.GetNotebook(context.Background(), "ghost")
	if err != nil || nb != nil {
		t.Fatalf("missing notebook = %+v, err = %v; want nil, nil", nb, err)
	}
}

func TestPoolResearchListMergesAccounts(t *testing.T) {
	a := &fakeAccount{
		listReportsFn: func(ctx context.Context) ([]rpcs.ResearchReport, string, error) {
			return []rpcs.ResearchReport{{Cid: "c_old", CreatedAt: 100}}, "cursor-a", nil
		},
	}
	b := &fakeAccount{
		listReportsFn: func(ctx context.Context) ([]rpcs.ResearchReport, string, error) {
			return []rpcs.ResearchReport{{Cid: "c_new", CreatedAt: 200}}, "cursor-b", nil
		},
	}
	pool := newAccountPool([]accountClient{a, b}, nil)

	reports, nextCursor, err := pool.ListResearchReportsPage(context.Background(), 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if nextCursor != "" {
		t.Fatalf("nextCursor = %q, want empty for multi-account", nextCursor)
	}
	if len(reports) != 2 || reports[0].Cid != "c_new" || reports[1].Cid != "c_old" {
		t.Fatalf("reports = %+v, want newest first", reports)
	}

	// Single account keeps the upstream cursor semantics.
	pool1 := newAccountPool([]accountClient{a}, nil)
	_, nextCursor, err = pool1.ListResearchReportsPage(context.Background(), 10, "in-cursor")
	if err != nil {
		t.Fatal(err)
	}
	if nextCursor != "cursor-a" {
		t.Fatalf("nextCursor = %q, want passthrough cursor-a", nextCursor)
	}
}

func TestPoolNotebookScopedNewChatUsesNotebookOwner(t *testing.T) {
	owner := &fakeAccount{
		getNotebookFn: func(ctx context.Context, id string) (*rpcs.Notebook, error) {
			return &rpcs.Notebook{ResourceName: "notebooks/" + id}, nil
		},
	}
	ownerCalled := false
	owner.generateContentFn = func(ctx context.Context, prompt string) (*types.ModelOutput, error) {
		ownerCalled = true
		return &types.ModelOutput{Text: "ok", Metadata: []string{"c_nb"}}, nil
	}
	other := &fakeAccount{
		generateContentFn: func(ctx context.Context, prompt string) (*types.ModelOutput, error) {
			t.Fatal("notebook-scoped chat must not rotate to a non-owner account")
			return nil, nil
		},
	}
	// Round-robin would pick `other` first; the notebook scope must override it.
	pool := newAccountPool([]accountClient{other, owner}, nil)

	out, err := pool.GenerateContent(context.Background(), "hi", nil, "nb-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ownerCalled || out.Text != "ok" {
		t.Fatalf("ownerCalled=%v out=%+v", ownerCalled, out)
	}

	// The created chat is pinned to the notebook owner's account.
	owner.sendMessageFn = func(ctx context.Context, md []string) (*types.ModelOutput, error) {
		return &types.ModelOutput{Text: "continued"}, nil
	}
	if _, err := pool.SendMessage(context.Background(), "next", []string{"c_nb"}, nil, ""); err != nil {
		t.Fatal(err)
	}
}

func TestPoolStreamFailoverAfterMetadataOnlyFrame(t *testing.T) {
	metaThenBroken := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			// Metadata-only frame: no delta reaches the caller, so failover
			// must still be allowed.
			cb(&types.ModelOutput{Metadata: []string{"c_x"}})
			return nil, fmt.Errorf("rate limited by server (HTTP 429)")
		},
	}
	healthy := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			return &types.ModelOutput{Text: "ok", Metadata: []string{"c_1"}}, nil
		},
	}
	pool := newAccountPool([]accountClient{metaThenBroken, healthy}, nil)

	out, err := pool.GenerateContentStream(context.Background(), "hi", nil, "", func(o *types.ModelOutput) {})
	if err != nil {
		t.Fatalf("failover after metadata-only frame should succeed: %v", err)
	}
	if out.Text != "ok" {
		t.Fatalf("output = %q", out.Text)
	}
}

func TestPoolGetNotebookProbeFailureIsNot404(t *testing.T) {
	broken := &fakeAccount{
		getNotebookFn: func(ctx context.Context, id string) (*rpcs.Notebook, error) {
			return nil, fmt.Errorf("init returned HTTP 429")
		},
	}
	pool := newAccountPool([]accountClient{broken}, nil)

	nb, err := pool.GetNotebook(context.Background(), "nb-1")
	if err == nil {
		t.Fatalf("probe failure must surface as error, got nb=%+v", nb)
	}
}

func TestPoolStreamDoesNotLeakFailedAccountMetadata(t *testing.T) {
	metaThenBroken := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			cb(&types.ModelOutput{Metadata: []string{"c_bad"}})
			return nil, fmt.Errorf("rate limited by server (HTTP 429)")
		},
	}
	healthy := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			cb(&types.ModelOutput{Metadata: []string{"c_good"}})
			cb(&types.ModelOutput{TextDelta: "hello"})
			return &types.ModelOutput{Text: "hello", Metadata: []string{"c_good"}}, nil
		},
	}
	pool := newAccountPool([]accountClient{metaThenBroken, healthy}, nil)

	var frames []*types.ModelOutput
	out, err := pool.GenerateContentStream(context.Background(), "hi", nil, "", func(o *types.ModelOutput) {
		frames = append(frames, o)
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "hello" {
		t.Fatalf("output = %q", out.Text)
	}
	for _, f := range frames {
		if len(f.Metadata) > 0 && f.Metadata[0] == "c_bad" {
			t.Fatalf("failed account's metadata leaked to caller: %+v", frames)
		}
	}
	// The winner's metadata frame must be delivered ahead of its delta.
	if len(frames) != 2 || len(frames[0].Metadata) == 0 || frames[0].Metadata[0] != "c_good" {
		t.Fatalf("frames = %+v, want winner metadata then delta", frames)
	}
}

func TestPoolStreamFlushesMetadataOnSuccessWithoutDelta(t *testing.T) {
	quiet := &fakeAccount{
		genStreamFn: func(ctx context.Context, cb client.StreamCallback) (*types.ModelOutput, error) {
			cb(&types.ModelOutput{Metadata: []string{"c_quiet"}})
			return &types.ModelOutput{Metadata: []string{"c_quiet"}}, nil
		},
	}
	pool := newAccountPool([]accountClient{quiet}, nil)

	var frames []*types.ModelOutput
	if _, err := pool.GenerateContentStream(context.Background(), "hi", nil, "", func(o *types.ModelOutput) {
		frames = append(frames, o)
	}); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || frames[0].Metadata[0] != "c_quiet" {
		t.Fatalf("frames = %+v, want the held metadata frame delivered", frames)
	}
}

func TestAccountLabel(t *testing.T) {
	pool := newAccountPool(
		[]accountClient{&fakeAccount{}, &fakeAccount{}},
		[]string{
			"/cookies/alice.json ($GEMINI_WEB_COOKIES_JSON_PATH)",
			"/cookies/bob.json ($GEMINI_WEB_COOKIES_JSON_PATH)",
		},
	)
	if got, want := pool.accountLabel(0), "account 1/2 (alice.json)"; got != want {
		t.Fatalf("accountLabel(0) = %q, want %q", got, want)
	}
	if got, want := pool.accountLabel(1), "account 2/2 (bob.json)"; got != want {
		t.Fatalf("accountLabel(1) = %q, want %q", got, want)
	}
	if got, want := shortSourceName("/tmp/foo (bar).json (--cookies-json)"), "foo (bar).json"; got != want {
		t.Fatalf("shortSourceName = %q, want %q", got, want)
	}

	pool.recordChat("c_1", 1)
	if got, want := pool.labelForChat("c_1"), "account 2/2 (bob.json)"; got != want {
		t.Fatalf("labelForChat = %q, want %q", got, want)
	}
}

func TestAccountSnapshotsLoginAndLastError(t *testing.T) {
	okAcc := &fakeAccount{}
	badAcc := &fakeAccount{
		initFn: func(ctx context.Context) error {
			return fmt.Errorf("GetNotebook rejected __Secure-1PSID=g.a000secret")
		},
	}
	pool := newAccountPool(
		[]accountClient{okAcc, badAcc},
		[]string{
			"/home/user/accounts/alice.json (--cookies-json)",
			"/home/user/accounts/bob.json (--cookies-json)",
		},
	)
	if err := pool.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	snaps := pool.AccountSnapshots()
	if len(snaps) != 2 {
		t.Fatalf("len = %d", len(snaps))
	}
	if snaps[0].Index != 1 || snaps[0].Name != "alice.json" || !snaps[0].LoggedIn {
		t.Fatalf("account 1 = %+v", snaps[0])
	}
	if snaps[1].Index != 2 || snaps[1].Name != "bob.json" || snaps[1].LoggedIn {
		t.Fatalf("account 2 = %+v", snaps[1])
	}
	if snaps[1].LastError == "" || snaps[1].LastErrorAt.IsZero() {
		t.Fatalf("account 2 missing last error: %+v", snaps[1])
	}
	if strings.Contains(snaps[1].LastError, "g.a000secret") || strings.Contains(snaps[1].Name, "/") {
		t.Fatalf("secret or path leaked: %+v", snaps[1])
	}
}
