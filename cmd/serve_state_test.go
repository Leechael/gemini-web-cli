package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCookiesJSONWithStateDirPriority(t *testing.T) {
	oldCookiesJSON := cookiesJSON
	t.Cleanup(func() { cookiesJSON = oldCookiesJSON })
	cookiesJSON = ""
	t.Setenv(envCookiesPath, filepath.Join(t.TempDir(), "env-cookies.json"))

	stateDir := t.TempDir()
	stateCookies := filepath.Join(stateDir, "cookies.json")
	if err := os.WriteFile(stateCookies, []byte(`{"cookies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	path, source := resolveCookiesJSONWithStateDir(stateDir)
	if path != stateCookies || source != "state-dir" {
		t.Fatalf("path/source = %q/%q, want %q/state-dir", path, source, stateCookies)
	}

	explicit := filepath.Join(t.TempDir(), "explicit.json")
	cookiesJSON = explicit
	path, source = resolveCookiesJSONWithStateDir(stateDir)
	if path != explicit || source != "--cookies-json" {
		t.Fatalf("explicit path/source = %q/%q", path, source)
	}
}
