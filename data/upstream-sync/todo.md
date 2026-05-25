# Upstream Sync TODO

Features to potentially implement, sourced from upstream commits/PRs.

## Pending

- [x] **3c601c8 — Image-to-image extraction gap** — Stage 6 adds `arr[0]["8"]` image-to-image extraction in `internal/types/types.go`.

- [x] **a2fd77f — URL hostname validation in download command** — already fixed by `isGoogleusercontentURL` in `cmd/download.go`.

- [x] **PR #309 — Veo/Lyria URL extraction** — Stage 5 adds protocol-layer extractor tests for the old video path and new `[12][8]["60"]` / `[12][0]["87"]` paths.

- [x] **PR #312 — Tier-aware model naming** — already implemented by dynamic discovery (`buildModelIDNameMapping`, `tierSuffixForCapacity`, `nameMatchesTierSuffix`) and preserved during GetUserStatus facade migration.

- [x] **PR #314 — Detect queueing frames for Veo/Lyria long jobs** — Stage 5 adds `QueueingError` when stream frames contain `Stream suspended` / `queueing=True`; real Veo/Lyria HAR can further narrow the exact path.

- [x] **PR #310 (partial) — Quota tracking** — implemented via `internal/client/quota.go` and exposed by `gemini-web-cli status`.

- [x] **PR #310 (partial) — Abuse status detection** — implemented via `internal/client/abuse.go` and exposed by `gemini-web-cli status`.

- [ ] **Stage 3 follow-up — DeleteChat (chat domain gap)**
  - Source: HAR doc `har-20260524.md` §4 (missing scenarios) + §6 (upstream parity table)
  - Upstream RPCs: `GzXR5e` `DELETE_CHAT_1` + `qWymEb` `DELETE_CHAT_2` (two-step delete in Python lib)
  - Block: HAR sample missing — current capture never deletes a chat. Need a dedicated capture: open Gemini web UI, delete one chat from list, record `delete-chat-NNNN.har`.
  - Where: new `protocol/rpcs/delete_chat.go` + business method `chat_delete.go` + CLI subcommand `chat delete <chatId>` (slots into existing `chat` cobra group from Stage 3)
  - Order: capture HAR → add protocol RPCs (2-step) → add business method (chain the two RPCs, both must succeed) → add CLI with confirmation prompt
  - User value: `gemini-web-cli chat delete <chatId>` to clean up unwanted chats from CLI

- [ ] **P9 Stage 2 — Template id wiring into StreamGenerate**
  - Block: missing HAR for complete `/images` template-to-generation flow.
  - Required capture: open `/images`, select a template, enter prompt, trigger image generation, save HAR.
  - Then: identify the StreamGenerate slot carrying template id and add a protocol option plus CLI generation flow.

- [ ] **Housekeeping dI8W6e completion**
  - Where: `internal/client/protocol/rpcs/housekeeping_log_event.go` currently keeps `dI8W6e` as a stub.
  - Block: needs a real payload sample with device id field semantics.
  - Risk: MyActivity iframe token plus device id context is not understood; main-page token may be rejected.

- [ ] **P9 Stage 2 — Banana Pro image generation (PR #304)**
  - Source: open upstream PR `#304`.
  - Block: PR is unmerged upstream; handle with P9 Stage 2 template id wiring after image generation HAR capture.
  - Change: add `pro_image` flag, set StreamGenerate slots 72=7 and 79=3, and append the Banana Pro message content extension.

## Ready to implement

(none — pending items above need decisions/HAR captures first)

## Completed

- [x] Dynamic model discovery + account status (upstream PR #280) — PR #5 feat/dynamic-model-discovery
- [x] Dynamic language + push_id extraction (upstream PR #280) — PR #4 feat/dynamic-push-id-language
