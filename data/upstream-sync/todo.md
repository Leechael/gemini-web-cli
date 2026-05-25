# Upstream Sync TODO

Features to potentially implement, sourced from upstream commits/PRs.

## Pending

- [ ] **3c601c8 — Image-to-image extraction gap**
  - Source: upstream commit `3c601c8` `fix: add image to image discovery to _parse_candidate`
  - Where: `internal/client/generate.go` `extractImages` (lines 629-681)
  - Change: also walk `arr[0]["8"][0]` (alongside existing `arr[7]`) to extract image-to-image generation results. Current code returns zero `Images` for image-to-image edits.

- [ ] **a2fd77f — URL hostname validation in download command**
  - Source: upstream commit `a2fd77f` `Potential fix for code scanning alert no. 5`
  - Where: `cmd/download.go:191-195`
  - Change: replace `strings.Contains(fileURL, "googleusercontent.com")` with `net/url` parse + host equals/HasSuffix check. Low real-world risk (only effect is appending `=s2048`) but closes static-analysis flag.

- [ ] **PR #304 — Banana Pro image generation** (waiting for upstream merge)
  - Source: open upstream PR `#304`
  - Where: `internal/client/generate.go` `buildInnerRequest` (line 221) and CLI flag plumbing
  - Change: add `pro_image` flag → extend inner_req_list to 80 elements, set slots 72=7, 79=3, append `[None,None,[None,None,None,None,None,None,[None,[1]]]]` to message_content.
  - Block: PR is unmerged upstream; wait for stable upstream consensus or implement independently if users request.

- [x] **PR #309 — Veo/Lyria URL extraction** — Stage 5 adds protocol-layer extractor tests for the old video path and new `[12][8]["60"]` / `[12][0]["87"]` paths.

- [x] **PR #312 — Tier-aware model naming** — already implemented by dynamic discovery (`buildModelIDNameMapping`, `tierSuffixForCapacity`, `nameMatchesTierSuffix`) and preserved during GetUserStatus facade migration.

- [x] **PR #314 — Detect queueing frames for Veo/Lyria long jobs** — Stage 5 adds `QueueingError` when stream frames contain `Stream suspended` / `queueing=True`; real Veo/Lyria HAR can further narrow the exact path.

- [ ] **PR #310 (partial) — Quota tracking**
  - Source: open upstream PR `#310`
  - New RPCs needed: `CHECK_GEMINI_QUOTA = "qpEbW"` and `CHECK_QUOTA = "aPya6c"` (note: Go currently uses these IDs for different purposes — verify mapping)
  - Payloads: `[[[1,11],[2,11],[6,11]]]` for Flash quota, `[[[1,4],[6,6],[1,15]]]` for Advanced (Pro + Flash Thinking) quota
  - Output: per-model remaining count, total, reset timestamp; expose via `gemini inspect` or new `gemini quota` subcommand
  - User value: see "X Pro requests left today, resets at HH:MM" before sending — currently zero visibility

- [ ] **PR #310 (partial) — Abuse status detection**
  - Source: open upstream PR `#310`
  - New RPC needed: `GET_ABUSE_STATUS = "GPRiHf"`, payload `[]`
  - Output: `is_clean` flag + status code + signal; expose as warning in `gemini inspect`
  - User value: early warning if Google has flagged the account

- [ ] **Stage 3 follow-up — DeleteChat (chat domain gap)**
  - Source: HAR doc `har-20260524.md` §4 (missing scenarios) + §6 (upstream parity table)
  - Upstream RPCs: `GzXR5e` `DELETE_CHAT_1` + `qWymEb` `DELETE_CHAT_2` (two-step delete in Python lib)
  - Block: HAR sample missing — current capture never deletes a chat. Need a dedicated capture: open Gemini web UI, delete one chat from list, record `delete-chat-NNNN.har`.
  - Where: new `protocol/rpcs/delete_chat.go` + business method `chat_delete.go` + CLI subcommand `chat delete <chatId>` (slots into existing `chat` cobra group from Stage 3)
  - Order: capture HAR → add protocol RPCs (2-step) → add business method (chain the two RPCs, both must succeed) → add CLI with confirmation prompt
  - User value: `gemini-web-cli chat delete <chatId>` to clean up unwanted chats from CLI

## Ready to implement

(none — pending items above need decisions/HAR captures first)

## Completed

- [x] Dynamic model discovery + account status (upstream PR #280) — PR #5 feat/dynamic-model-discovery
- [x] Dynamic language + push_id extraction (upstream PR #280) — PR #4 feat/dynamic-push-id-language
