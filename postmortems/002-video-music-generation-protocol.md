# Video & Music Generation Protocol — Debugging Postmortem

## Summary

Adding video and music generation support required discovering that StreamGenerate and ReadChat use fundamentally different response structures for the same data. StreamGenerate uses long positional arrays (index 59/86), while ReadChat compacts them into a short array with a trailing dict (keys "60"/"87").

## Timeline of Issues

### 1. Model Naming Update — Upstream Breaking Change

**Symptom**: Models like `gemini-3.0-flash` and `gemini-3.1-pro` stopped working (error 1052).

**Root Cause**: Upstream Python library renamed all models:
- `gemini-3.0-flash` → `gemini-3-flash`
- `gemini-3.1-pro` → `gemini-3-pro` (different header hash too)
- `gemini-2.x` models removed entirely
- New tiers added: `plus` (tier 4) and `advanced` (tier 2)

**Fix**: Replace entire model list. The tier number is the last element in the `x-goog-ext-525001261-jspb` header array.

**Lesson**: Model header hashes and tier numbers change across versions. Always sync from upstream constants.

### 2. Card URL Placeholders Not Stripped

**Symptom**: Response text contained `http://googleusercontent.com/video_gen_chip/0` or `http://googleusercontent.com/generated_video_content/0` as visible text.

**Root Cause**: The existing cleanup only stripped `card_content/` prefixes (for images). Video and music generation use different placeholder URLs.

**Known placeholder patterns**:
```
http://googleusercontent.com/card_content/N          — images
http://googleusercontent.com/video_gen_chip/N        — video (generating)
http://googleusercontent.com/generated_video_content/N — video (ready)
http://googleusercontent.com/generated_music_content/N — music (ready)
http://googleusercontent.com/generated_media_content/N — media (generic)
```

**Fix**: Strip all `http://googleusercontent.com/` prefixed lines from response text.

**Lesson**: Each generation type has its own placeholder URL pattern. The placeholder changes from `*_chip` (generating) to `generated_*_content` (ready).

### 3. StreamGenerate vs ReadChat — Different Array Structures

**Symptom**: Video/music URLs extracted correctly during `ask` (StreamGenerate) but `get` (ReadChat) returned no videos/media despite raw response containing them.

**Root Cause**: The two endpoints structure `candidate[12]` differently:

**StreamGenerate** — long positional array (87+ elements):
```
candidate[12][59]  → video data
candidate[12][86]  → media/music data
```

**ReadChat** — short array (typically 9 elements) with dict at end:
```
candidate[12] = [null, webImgs, ..., null, genImgs, imgData, {"32":..., "60":..., "87":...}]
candidate[12][8]  → dict with string keys
  key "60" → video data (maps to StreamGenerate index 59)
  key "87" → media/music data (maps to StreamGenerate index 86)
  key "32" → system instructions/memory
```

**Fix**: Extraction functions try the direct index path first, then scan array elements for dicts with the corresponding key.

**Lesson**: The wire format is a sparse protobuf-like structure. StreamGenerate uses the raw positional format (nulls for empty slots), while ReadChat uses a compact representation where high-index fields are folded into a dict. The dict keys are string-encoded integers that roughly correspond to the positional indices (+1 offset observed: 59→"60", 86→"87").

### 4. RPC Response Not Found — ReadChat Parsing Failure

**Symptom**: `get c_xxx` returned `Error: RPC response for hNvQHb not found` on some requests.

**Root Cause**: Network/proxy issue (TLS handshake timeout) caused a non-standard response body that didn't contain the expected `wrb.fr` envelope.

**Fix**: Added `SilenceUsage: true` to avoid confusing error-on-error output. The RPC ID `hNvQHb` itself is still correct.

**Lesson**: Not every parsing failure is a protocol change. Network errors can produce partial responses that fail RPC body extraction.

## Protocol Reference: Video/Music Wire Format

### Request

Video and music generation use the **same request format as normal chat**. No special flags are needed (unlike deep research). The model and prompt determine what gets generated.

### StreamGenerate Response — Video at candidate[12][59]

```
candidate[12][59][0][0][0][0][7] = [thumbnail_url, download_url, thumbnail_url_alt]
```

Full navigation:
```
candidate[12][59]          → top-level video wrapper
  [0]                      → first video group
    [0]                    → video entries
      [0]                  → first entry
        [0]                → video element: [null, 2, "video.mp4", null, null, "$hash", null, [urls]]
          [7]              → URL array
            [0] = thumbnail (lh3.googleusercontent.com)
            [1] = download  (contribution.usercontent.google.com/download?...)
            [2] = thumbnail alt
```

### StreamGenerate Response — Music/Media at candidate[12][86]

```
candidate[12][86][0][1][7] = [thumb, mp3_download_url, thumb_alt]   — MP3
candidate[12][86][1][1][7] = [thumb, mp4_download_url, thumb_alt]   — MP4 (if available)
```

Music generation may produce MP3 only, or both MP3+MP4.

### ReadChat Response — Dict-Based Format

```
candidate[12][last_element] = {
  "60": video data (same nested structure as StreamGenerate [59]),
  "87": media data (same nested structure as StreamGenerate [86]),
  "32": system instructions / memory context,
  "7":  [0],
  "8":  []
}
```

The dict element is always the last element in the `candidate[12]` array.

### Download URLs

Video/music download URLs use Google's content delivery:
```
https://contribution.usercontent.google.com/download?c=<base64>&filename=<name>&opi=103135050
```

Thumbnails use:
```
https://lh3.googleusercontent.com/gg/<path>=m140
```

### HTTP 206 Polling

When a video/music is still being generated, the download URL returns **HTTP 206** (Partial Content) instead of 200. The client should poll every ~10 seconds until it receives 200 with the complete file.

This differs from deep research which uses a separate RPC-based progress mechanism.

### Generation Lifecycle

1. User sends prompt via StreamGenerate
2. Server returns text with placeholder URL (`video_gen_chip/0` or similar)
3. Generation happens server-side (can take minutes)
4. Re-fetching the chat via ReadChat shows:
   - Updated text (placeholder changes to `generated_video_content/0`)
   - Actual download URLs in the candidate[12] dict structure
5. Download URL returns 206 while generating, 200 when ready
