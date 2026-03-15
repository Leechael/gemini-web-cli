# Deep Research Protocol — Debugging Postmortem

## Summary

Implementing `research send` required 5 iterations to get the full plan→confirm→start flow working. Each fix revealed a deeper protocol assumption that differed from normal chat requests.

## Timeline of Issues

### 1. HTTP/1.1 vs HTTP/2 — Server Rejects Deep Research Flags

**Symptom**: Server returned the Deep Research welcome page instead of creating a plan, despite correct inner_req_list flags (`[68]=2`, `[49]=1`, etc.).

**Root Cause**: Go's `net/http` defaults to HTTP/1.1. Python's `httpx` uses `http2=True`. The Gemini server only processes deep research flags over HTTP/2 connections.

**Fix**: `transport.ForceAttemptHTTP2 = true`

**Lesson**: Protocol version matters. The server may silently downgrade behavior on HTTP/1.1 without returning an error.

### 2. Metadata Format — 3 vs 10 Elements

**Symptom**: Even with HTTP/2, some requests were treated as new conversations.

**Root Cause**: Python's `ChatSession.__metadata` is always a 10-element array:
```
["", "", "", null, null, null, null, null, null, ""]
```
Go initially used `["", "", ""]` (3 elements) for new chats.

**Fix**: Default metadata is 10 elements: `["", "", "", nil*6, ""]` matching browser HAR captures.

**Lesson**: Array length matters in the protobuf-like wire format. Even null padding positions carry semantic meaning.

### 3. Missing rcid in Metadata — Confirm Step Ignored

**Symptom**: Step 2 (confirm) was sent with `[cid, rid]` (2 elements), but server expected `[cid, rid, rcid, ...]` (10 elements with rcid at position [2]).

**Root Cause**: Stream response `content[1]` only has `[cid, rid]` (2 elements). Python's `ChatSession` separately tracks `chat.rcid` from `candidate_data[0]` and merges it into position [2] of the metadata array. Go wasn't capturing rcid from the candidate.

**Fix**: In `parseEnvelope`, after extracting rcid from the candidate, inject it into `metadata[2]` and pad to 10 elements.

**Lesson**: Metadata is assembled from multiple sources:
- `content[1]` → cid, rid (positions 0, 1)
- `candidate[0]` → rcid (position 2)
- Completion frame → context string (position 9)

Python's `ChatSession.metadata` setter uses **merge semantics** (only update non-None positions), not replacement.

### 4. Context String Location — content[2]["26"] Not content[25]

**Symptom**: Step 2 received metadata without context string at position [9]. The confirm request was sent without the context token, causing the server to treat it as a new deep research session.

**Root Cause**: Normal chat completion uses `content[25]` (a string) as the context token. Deep research uses a **different completion frame format**: `content[2]` is a dict `{"26": "AwAAAA...", "44": false}` where key `"26"` holds the context string.

**Discovery**: HAR analysis of `data/sample01.har` entry[54] showed the context string at position 9515 in the raw response, inside `{"26":"AwAAAA..."}`. The frame structure is:
```json
["wrb.fr", null, "[null, [\"cid\", \"rid\"], {\"26\": \"context_str\", \"44\": false}]"]
```

**Fix**: Check both `content[25]` (normal) and `content[2]` as dict with key `"26"` (deep research) for the context string.

**Lesson**: The "protobuf" format uses positional arrays AND dict keys interchangeably. Completion signaling has at least two distinct wire formats depending on the feature.

### 5. Plan Data Extraction — collectStreamResult vs Direct Callback

**Symptom**: Plan title, steps, ETA, and confirm_prompt were extracted correctly in `parseEnvelope` (verified by debug logging) but didn't reach the research flow code.

**Root Cause**: `deepResearchGenerate` used raw `streamGenerate` with `last = out` callback, keeping only the last frame. The plan data appeared in intermediate frames but was overwritten by the final frame (which didn't have plan data). `collectStreamResult` (used by `GenerateContent`) properly accumulates plan/images across frames, but `deepResearchGenerate` didn't use it.

**Fix**: Change `deepResearchGenerate` to use `collectStreamResult`.

**Lesson**: For multi-frame features (images, plans, metadata), always use the accumulating collector. Direct "last frame" pattern only works for text-only responses.

### 6. Incremental Stream Parsing — Deep Research Streams Don't Close

**Symptom**: `io.ReadAll` blocked for 300 seconds on deep research step 1, then timed out.

**Root Cause**: Normal chat streams close after sending all frames (including the `content[25]` completion marker). Deep research streams send the plan frames then keep the connection open (no `content[25]`). The completion is signaled by `content[2]["26"]` in a special frame, but the TCP connection stays alive.

**Fix**: Rewrite `parseStreamResponse` to read incrementally (`body.Read` in a loop), parse frames as they arrive, and stop early when `Done` is detected.

**Lesson**: Streaming endpoints may have different connection lifecycle depending on the feature. Always process incrementally, never `io.ReadAll` for streaming responses.

## Protocol Reference: Deep Research Wire Format

### Request (inner_req_list, 69 elements)

Key deep research indices (beyond normal chat):
```
[3]  = "!" + url_safe_token(2600)   # random per-request
[4]  = hex_uuid (32 chars)          # random per-request
[49] = 1                            # deep research flag
[54] = [[[[[1]]]]]                  # deep research marker
[55] = [[1]]                        # deep research marker
[68] = 2                            # mode: 2=deep_research, 1=normal
```

Default metadata (10 elements):
```
[0] = cid or ""
[1] = rid or ""
[2] = rcid or ""       ← from candidate[0], NOT from stream metadata
[3-8] = null
[9] = context_str or "" ← from completion frame content[2]["26"]
```

### Response Completion Frames

**Normal chat**: content[25] is a string (the context token). The stream then closes.

**Deep research**: content[25] is null. Instead, a separate frame has:
```json
[null, ["cid", "rid"], {"26": "context_string", "44": false}]
```
The stream may stay open after this frame.

### Other Dict-Keyed Frames in Deep Research

During streaming, several dict-keyed frames appear at `content[2]`:
- `{"7": [...]}` — plan generation status
- `{"11": ["title"]}` — chat title assignment
- `{"18": "rid"}` — reply ID assignment
- `{"21": ["session_token"]}` — session binding
- `{"26": "context_str"}` — **completion with context** (critical for metadata[9])
- `{"28": [[...], "model_id"]}` — model usage info
- `{"44": false}` — present in all dict frames (unknown purpose)

### Plan Data Structure (in candidate dict)

Located recursively inside the candidate array in a dict at key `"56"` or `"57"`:
```
payload[0]    = title (string)
payload[1]    = steps array, each: [index, label, body]
payload[2]    = eta_text (string)
payload[3][0] = confirm_prompt (string) — MUST be used for step 2
payload[4][0] = confirmation_url
payload[5]    = modify_payload
```

The dict also contains:
- `"58"` — rate limit / quota info
- `"70"` — state integer (2=plan_ready, 3=research_running)

### Preflight RPCs (before each StreamGenerate)

```
ESY5D   (BARD_ACTIVITY)        — always
L5adhe  (DEEP_RESEARCH_PREFS)  — feature state, popup state
ku4Jyf  (DEEP_RESEARCH_BOOTSTRAP) — language/model init
qpEbW   (DEEP_RESEARCH_MODEL_STATE) — with cid
aPya6c  (DEEP_RESEARCH_CAPS)   — with cid
PCck7e  (DEEP_RESEARCH_ACK)    — with rid (after each StreamGenerate)
```

All are best-effort (errors logged, not fatal).

### Two-Step Flow

1. **Step 1**: Send prompt with deep_research=true → get cid, rid, rcid, context_str, plan data
2. **Step 2**: Send `plan.confirm_prompt` with metadata=[cid, rid, rcid, ..., context_str] → research starts

The `confirm_prompt` MUST come from the plan data (key `"56"` or `"57"` at `payload[3][0]`). Using a hardcoded "开始研究" may work for Chinese responses but fails for English responses where the server returns "Start research".
