package cookies

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// CookieJar holds loaded cookies and their metadata.
type CookieJar struct {
	Cookies map[string]string
	Meta    map[string]CookieMeta
}

// CookieMeta holds expiry information for a cookie.
type CookieMeta struct {
	ExpiresRaw   any    `json:"expires_raw"`
	ExpiresEpoch *int64 `json:"expires_epoch"`
	ExpiresISO   string `json:"expires_iso"`
}

// Load reads a cookie JSON file supporting multiple formats:
//   - {name: value, ...}
//   - {"cookies": {name: value, ...}}
//   - {"cookies": [{name, value, ...}, ...]}
//   - [{name, value, ...}, ...]
func Load(path string) (*CookieJar, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cookie file: %w", err)
	}

	jar := &CookieJar{
		Cookies: make(map[string]string),
		Meta:    make(map[string]CookieMeta),
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing cookie JSON: %w", err)
	}

	switch v := raw.(type) {
	case map[string]any:
		// Check for {"cookies": ...} wrapper
		if inner, ok := v["cookies"]; ok {
			switch ci := inner.(type) {
			case map[string]any:
				// {"cookies": {name: value}}
				for k, val := range ci {
					if s, ok := val.(string); ok && s != "" {
						jar.upsert(k, s, nil)
					}
				}
				return jar, nil
			case []any:
				// {"cookies": [{name, value}, ...]}
				for _, item := range ci {
					if obj, ok := item.(map[string]any); ok {
						jar.handleCookieObj(obj)
					}
				}
				if len(jar.Cookies) > 0 {
					return jar, nil
				}
			}
		}
		// Flat {name: value}
		allStrings := true
		for _, val := range v {
			if _, ok := val.(string); !ok {
				allStrings = false
				break
			}
		}
		if allStrings {
			for k, val := range v {
				jar.upsert(k, val.(string), nil)
			}
			return jar, nil
		}
	case []any:
		// [{name, value, ...}, ...]
		for _, item := range v {
			if obj, ok := item.(map[string]any); ok {
				jar.handleCookieObj(obj)
			}
		}
		if len(jar.Cookies) > 0 {
			return jar, nil
		}
	}

	return nil, fmt.Errorf("unsupported cookie format; expected {name:value}, {\"cookies\":[...]} or [...]")
}

func (j *CookieJar) upsert(name, value string, expiresRaw any) {
	if name == "" || value == "" {
		return
	}
	j.Cookies[name] = value

	m := CookieMeta{ExpiresRaw: expiresRaw}
	if epoch := parseExpiry(expiresRaw); epoch != nil {
		m.ExpiresEpoch = epoch
		t := time.Unix(*epoch, 0).UTC()
		m.ExpiresISO = t.Format(time.RFC3339)
	}
	j.Meta[name] = m
}

func (j *CookieJar) handleCookieObj(obj map[string]any) {
	name, _ := obj["name"].(string)
	value, _ := obj["value"].(string)
	if name == "" || value == "" {
		return
	}

	var expiresRaw any
	for _, key := range []string{"expirationDate", "expires", "expiry", "expiresDate"} {
		if v, ok := obj[key]; ok {
			expiresRaw = v
			break
		}
	}
	j.upsert(name, value, expiresRaw)
}

func parseExpiry(v any) *int64 {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case float64:
		e := int64(val)
		return &e
	case int64:
		return &val
	case string:
		if val == "" {
			return nil
		}
		// Try parsing as number
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			e := int64(f)
			return &e
		}
		// Try ISO 8601
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z"} {
			if t, err := time.Parse(layout, val); err == nil {
				e := t.Unix()
				return &e
			}
		}
	}
	return nil
}

// Persist writes updated cookies back to the JSON file.
func Persist(path string, original map[string]string, updated map[string]string, verbose bool) error {
	merged := make(map[string]string)
	for k, v := range original {
		merged[k] = v
	}
	for k, v := range updated {
		if v != "" {
			merged[k] = v
		}
	}

	// Check if anything changed
	if len(merged) == len(original) {
		same := true
		for k, v := range merged {
			if original[k] != v {
				same = false
				break
			}
		}
		if same {
			return nil
		}
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sorted := make(map[string]string)
	for _, k := range keys {
		sorted[k] = merged[k]
	}

	payload := map[string]any{
		"updated_at": time.Now().UTC().Format(time.RFC3339),
		"cookies":    sorted,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
