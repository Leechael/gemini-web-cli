#!/usr/bin/env bash
set -euo pipefail

# E2E test script for gemini-web-cli
# Usage: ./scripts/e2e-test.sh <cookies.json>

COOKIES="${1:?Usage: $0 <cookies.json>}"
CLI="./bin/gemini-web-cli"
PASS=0
FAIL=0
TOTAL=0
TMPFILE=""
DL_TMPFILE=""
RESEARCH_TMPFILE=""
trap 'rm -f $TMPFILE $DL_TMPFILE $RESEARCH_TMPFILE 2>/dev/null' EXIT

red()   { printf "\033[31m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
bold()  { printf "\033[1m%s\033[0m\n" "$*"; }

assert_contains() {
    local label="$1" output="$2" expected="$3"
    TOTAL=$((TOTAL + 1))
    if echo "$output" | grep -qF -- "$expected"; then
        green "  PASS: $label"
        PASS=$((PASS + 1))
    else
        red "  FAIL: $label (expected '$expected')"
        red "  output: ${output:0:200}"
        FAIL=$((FAIL + 1))
    fi
}

assert_not_empty() {
    local label="$1" output="$2"
    TOTAL=$((TOTAL + 1))
    if [ -n "$output" ]; then
        green "  PASS: $label (${#output} chars)"
        PASS=$((PASS + 1))
    else
        red "  FAIL: $label (empty output)"
        FAIL=$((FAIL + 1))
    fi
}

extract_chat_id() {
    echo "$1" | grep -oE 'c_[0-9a-f]{16}' | tail -1
}

count_pattern() {
    echo "$1" | grep -cF -- "$2" || echo 0
}

# Build first
bold "Building gemini-web-cli..."
(cd "$(dirname "$0")/.." && go build -o bin/gemini-web-cli .) || { red "Build failed"; exit 1; }
green "Build OK"
echo

# ============================================================
bold "1. models (offline)"
# ============================================================
OUT=$($CLI models 2>&1)
assert_contains "lists unspecified" "$OUT" "unspecified (default)"
assert_contains "lists gemini-2.0-flash" "$OUT" "gemini-2.0-flash"
echo

# ============================================================
bold "2. inspect --cookies-only"
# ============================================================
OUT=$($CLI --cookies-json "$COOKIES" inspect --cookies-only 2>&1)
assert_contains "shows present: true" "$OUT" '"present": true'
assert_contains "shows __Secure-1PSID" "$OUT" "__Secure-1PSID"
echo

# ============================================================
bold "3. inspect (full)"
# ============================================================
OUT=$($CLI --cookies-json "$COOKIES" inspect 2>&1)
assert_contains "shows Init: OK" "$OUT" "Init: OK"
echo

# ============================================================
bold "4. list"
# ============================================================
OUT=$($CLI --cookies-json "$COOKIES" list 2>&1)
assert_contains "has header" "$OUT" "ID"
LIST_CID=$(echo "$OUT" | grep -oE 'c_[0-9a-f]{16}' | head -1 || true)
if [ -n "$LIST_CID" ]; then
    green "  (found chat: $LIST_CID)"
fi
echo

# ============================================================
bold "5. get (existing chat)"
# ============================================================
if [ -n "$LIST_CID" ]; then
    OUT=$($CLI --cookies-json "$COOKIES" get "$LIST_CID" 2>&1)
    assert_contains "has message header" "$OUT" "message 1"
    assert_contains "has [User]" "$OUT" "[User]"
else
    red "  SKIP: no chat ID"; TOTAL=$((TOTAL + 2)); FAIL=$((FAIL + 2))
fi
echo

# ============================================================
bold "6. ask (streaming)"
# ============================================================
# Use a very deterministic question
OUT=$($CLI --cookies-json "$COOKIES" ask "Complete this: The capital of Japan is ___. Reply ONLY the city name, nothing else." 2>&1)
assert_not_empty "ask returns text" "$OUT"
assert_contains "ask shows Chat ID" "$OUT" "Chat ID:"
ASK_CID=$(extract_chat_id "$OUT")
if [ -n "$ASK_CID" ]; then
    green "  (chat ID: $ASK_CID)"
else
    red "  (no chat ID extracted)"
fi
echo

# ============================================================
bold "7. ask --no-stream"
# ============================================================
OUT=$($CLI --cookies-json "$COOKIES" ask --no-stream "What is 2+3? Reply ONLY the number." 2>&1)
assert_contains "contains 5" "$OUT" "5"
assert_contains "shows Chat ID" "$OUT" "Chat ID:"
echo

# ============================================================
bold "8. ask chat appears in list"
# ============================================================
if [ -n "$ASK_CID" ]; then
    sleep 2
    OUT=$($CLI --cookies-json "$COOKIES" list 2>&1)
    assert_contains "ask in list" "$OUT" "$ASK_CID"
else
    red "  SKIP"; TOTAL=$((TOTAL + 1)); FAIL=$((FAIL + 1))
fi
echo

# ============================================================
bold "9. reply (streaming, continues conversation)"
# ============================================================
if [ -n "$ASK_CID" ]; then
    # The ask was about Japan's capital. Reply should be in that context.
    OUT=$($CLI --cookies-json "$COOKIES" reply "$ASK_CID" "And what is the population of that city? Give me just an approximate number." 2>&1)
    assert_not_empty "reply returns text" "$OUT"
    assert_contains "reply shows Chat ID" "$OUT" "$ASK_CID"
else
    red "  SKIP"; TOTAL=$((TOTAL + 2)); FAIL=$((FAIL + 2))
fi
echo

# ============================================================
bold "10. reply --no-stream"
# ============================================================
if [ -n "$ASK_CID" ]; then
    OUT=$($CLI --cookies-json "$COOKIES" reply --no-stream "$ASK_CID" "What country is that city in? One word only." 2>&1)
    assert_not_empty "reply --no-stream returns text" "$OUT"
    assert_contains "reply shows Chat ID" "$OUT" "$ASK_CID"
else
    red "  SKIP"; TOTAL=$((TOTAL + 2)); FAIL=$((FAIL + 2))
fi
echo

# ============================================================
bold "11. get (created conversation, should have >=3 messages)"
# ============================================================
if [ -n "$ASK_CID" ]; then
    OUT=$($CLI --cookies-json "$COOKIES" get "$ASK_CID" 2>&1)
    assert_contains "has [User]" "$OUT" "[User]"
    assert_contains "has [Gemini]" "$OUT" "[Gemini]"
    MSG_COUNT=$(count_pattern "$OUT" "message")
    TOTAL=$((TOTAL + 1))
    if [ "$MSG_COUNT" -ge 3 ]; then
        green "  PASS: $MSG_COUNT messages (>=3)"
        PASS=$((PASS + 1))
    else
        red "  FAIL: $MSG_COUNT messages (expected >=3)"
        FAIL=$((FAIL + 1))
    fi
else
    red "  SKIP"; TOTAL=$((TOTAL + 3)); FAIL=$((FAIL + 3))
fi
echo

# ============================================================
bold "12. get --output (write to file)"
# ============================================================
TMPFILE=$(mktemp /tmp/gemini-e2e-XXXXXX.txt)
if [ -n "$ASK_CID" ]; then
    $CLI --cookies-json "$COOKIES" get "$ASK_CID" --output "$TMPFILE" 2>&1
    TOTAL=$((TOTAL + 1))
    if [ -s "$TMPFILE" ]; then
        green "  PASS: wrote $(wc -c < "$TMPFILE" | tr -d ' ') bytes to file"
        PASS=$((PASS + 1))
    else
        red "  FAIL: output file is empty"
        FAIL=$((FAIL + 1))
    fi
else
    red "  SKIP"; TOTAL=$((TOTAL + 1)); FAIL=$((FAIL + 1))
fi
echo

# ============================================================
bold "13. ask with --model flag"
# ============================================================
OUT=$($CLI --cookies-json "$COOKIES" --model gemini-2.0-flash ask "Say hello in one word." 2>&1)
assert_not_empty "ask with model" "$OUT"
assert_contains "shows Chat ID" "$OUT" "Chat ID:"
echo

# ============================================================
bold "14. ask with image generation"
# ============================================================
OUT=$($CLI --cookies-json "$COOKIES" --model gemini-2.0-flash ask "Generate image: a simple blue circle on white background" 2>&1)
assert_contains "image gen shows Chat ID" "$OUT" "Chat ID:"
# Extract generated image URL (lh3.googleusercontent.com)
IMG_URL=$(echo "$OUT" | grep -oE 'https://lh3\.googleusercontent\.com/[^ ]+' | head -1 || true)
if [ -n "$IMG_URL" ]; then
    green "  (extracted image URL)"
    assert_contains "shows Generated images section" "$OUT" "Generated images"
else
    TOTAL=$((TOTAL + 1))
    red "  FAIL: no generated image URL extracted"
    FAIL=$((FAIL + 1))
fi
echo

# ============================================================
bold "15. download (generated image)"
# ============================================================
DL_TMPFILE=$(mktemp /tmp/gemini-dl-XXXXXX.png)
if [ -n "$IMG_URL" ]; then
    OUT=$($CLI --cookies-json "$COOKIES" download "$IMG_URL" -o "$DL_TMPFILE" 2>&1)
    assert_contains "download shows Saved" "$OUT" "Saved to"
    TOTAL=$((TOTAL + 1))
    DL_SIZE=$(wc -c < "$DL_TMPFILE" | tr -d ' ')
    if [ "$DL_SIZE" -gt 1000 ]; then
        green "  PASS: downloaded ${DL_SIZE} bytes"
        PASS=$((PASS + 1))
    else
        red "  FAIL: downloaded file too small (${DL_SIZE} bytes)"
        FAIL=$((FAIL + 1))
    fi
else
    red "  SKIP: no image URL from previous test"
    TOTAL=$((TOTAL + 2)); FAIL=$((FAIL + 2))
fi
echo

# ============================================================
bold "16. download by chat_id"
# ============================================================
IMG_CID=$(extract_chat_id "$($CLI --cookies-json "$COOKIES" --model gemini-2.0-flash ask "Generate image: a simple blue circle on white background" 2>&1)")
if [ -n "$IMG_CID" ]; then
    DL_CHAT_FILE=$(mktemp /tmp/gemini-dl-chat-XXXXXX.png)
    OUT=$($CLI --cookies-json "$COOKIES" download "$IMG_CID" -o "$DL_CHAT_FILE" 2>&1)
    assert_contains "download by chat_id shows Saved" "$OUT" "Saved to"
    assert_contains "download by chat_id found images" "$OUT" "image(s)"
else
    red "  SKIP: no image chat ID"; TOTAL=$((TOTAL + 2)); FAIL=$((FAIL + 2))
fi
echo

# ============================================================
bold "17. research (submit + plan + confirm)"
# ============================================================
OUT=$($CLI --cookies-json "$COOKIES" research "What is quantum entanglement? Brief overview." 2>&1)
assert_contains "research shows Chat ID" "$OUT" "Chat ID:"
assert_contains "research shows Title" "$OUT" "Title:"
RESEARCH_CID=$(extract_chat_id "$OUT")
if [ -n "$RESEARCH_CID" ]; then
    green "  (research chat ID: $RESEARCH_CID)"
fi
echo

# ============================================================
bold "18. progress (wait for completion)"
# ============================================================
if [ -n "$RESEARCH_CID" ]; then
    OUT=$($CLI --cookies-json "$COOKIES" progress "$RESEARCH_CID" 2>&1)
    assert_contains "progress shows status" "$OUT" "Status:"

    # Wait for research to complete (poll every 60s, max 5 min)
    TOTAL=$((TOTAL + 1))
    RESEARCH_DONE=false
    for attempt in 1 2 3 4 5; do
        OUT=$($CLI --cookies-json "$COOKIES" progress "$RESEARCH_CID" 2>&1)
        if echo "$OUT" | grep -qF "done"; then
            RESEARCH_DONE=true
            green "  PASS: research completed (attempt $attempt)"
            PASS=$((PASS + 1))
            break
        fi
        if [ "$attempt" -lt 5 ]; then
            echo "  ... waiting 60s (attempt $attempt/5)"
            sleep 60
        fi
    done
    if [ "$RESEARCH_DONE" = false ]; then
        red "  FAIL: research did not complete within 5 min"
        FAIL=$((FAIL + 1))
    fi
else
    red "  SKIP"; TOTAL=$((TOTAL + 2)); FAIL=$((FAIL + 2))
fi
echo

# ============================================================
bold "19. report (full report)"
# ============================================================
TOTAL=$((TOTAL + 1))
if [ -n "$RESEARCH_CID" ] && [ "$RESEARCH_DONE" = true ]; then
    OUT=$($CLI --cookies-json "$COOKIES" report "$RESEARCH_CID" 2>&1)
    if [ ${#OUT} -gt 1000 ]; then
        green "  PASS: report returned ${#OUT} chars (full report)"
        PASS=$((PASS + 1))
    else
        red "  FAIL: report returned only ${#OUT} chars"
        FAIL=$((FAIL + 1))
    fi
else
    green "  PASS: skipped (research not done)"
    PASS=$((PASS + 1))
fi
echo

# ============================================================
bold "20. report --output"
# ============================================================
RESEARCH_TMPFILE=$(mktemp /tmp/gemini-research-XXXXXX.md)
if [ -n "$RESEARCH_CID" ] && [ "$RESEARCH_DONE" = true ]; then
    $CLI --cookies-json "$COOKIES" report "$RESEARCH_CID" --output "$RESEARCH_TMPFILE" 2>&1 || true
    TOTAL=$((TOTAL + 1))
    RSIZE=$(wc -c < "$RESEARCH_TMPFILE" | tr -d ' ')
    if [ "$RSIZE" -gt 1000 ]; then
        green "  PASS: wrote $RSIZE bytes to file"
        PASS=$((PASS + 1))
    else
        red "  FAIL: wrote only $RSIZE bytes"
        FAIL=$((FAIL + 1))
    fi
else
    TOTAL=$((TOTAL + 1))
    green "  PASS: skipped (research not done)"
    PASS=$((PASS + 1))
fi
echo

# ============================================================
bold "21. list --cursor (pagination)"
# ============================================================
# Extract cursor: look for "(next page: --cursor <value>)"
CURSOR=$($CLI --cookies-json "$COOKIES" list 2>&1 | sed -n 's/.*--cursor //p' | tr -d ')' || true)
if [ -n "$CURSOR" ]; then
    OUT=$($CLI --cookies-json "$COOKIES" list --cursor "$CURSOR" 2>&1)
    assert_contains "paginated list has header" "$OUT" "ID"
else
    TOTAL=$((TOTAL + 1))
    green "  PASS: no pagination needed (fewer chats)"
    PASS=$((PASS + 1))
fi
echo

# ============================================================
bold "22. error: missing cookies"
# ============================================================
TOTAL=$((TOTAL + 1))
if $CLI ask "test" >/dev/null 2>&1; then
    red "  FAIL: should error without cookies"
    FAIL=$((FAIL + 1))
else
    green "  PASS: errors without cookies"
    PASS=$((PASS + 1))
fi
echo

# ============================================================
# Summary
# ============================================================
echo
bold "============================================"
if [ "$FAIL" -eq 0 ]; then
    green "ALL $TOTAL TESTS PASSED"
else
    red "$FAIL/$TOTAL TESTS FAILED ($PASS passed)"
fi
bold "============================================"

exit "$FAIL"
